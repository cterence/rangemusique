package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/michiwend/gomusicbrainz"
	"go.senan.xyz/taglib"
)

type trackElements struct {
	artist      string
	album       string
	title       string
	trackNumber string
	year        string
}

const (
	unknownArtist      = "Unknown Artist"
	unknownAlbum       = "Unknown Album"
	unknownTitle       = "Unknown Title"
	unknownTrackNumber = "0"
	unknownYear        = "0"
)

func Run(ctx context.Context, cfg Config) error {
	err := initApp(&cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize app: %w", err)
	}

	musicFilePaths := []string{}
	err = filepath.WalkDir(cfg.InputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		buf, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		if !filetype.IsAudio(buf) {
			slog.Debug("skipping non-audio file", "path", path)
			return nil
		}

		musicFilePaths = append(musicFilePaths, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to read input directory files: %w", err)
	}

	if len(musicFilePaths) == 0 {
		slog.Info("no music files found in input directory, exiting")
		return nil
	}

	for _, musicFilePath := range musicFilePaths {
		musicFileName := path.Base(musicFilePath)

		tags, err := taglib.ReadTags(musicFilePath)
		if err != nil {
			return fmt.Errorf("failed to read tags from file %s: %w", musicFileName, err)
		}
		slog.Debug("tags", "musicFileName", musicFileName, "tags", tags)

		trackElements, err := getTrackElementsFromTags(tags)
		if err != nil {
			slog.Warn("failed to extract track elements from file, skipping", "musicFileName", musicFileName, "error", err)
			continue
		}

		if trackElements.year == unknownYear && trackElements.artist != unknownArtist && trackElements.album != unknownAlbum {
			releaseYear, err := getReleaseYearFromMusicBrainz(cfg.mbClient, trackElements.artist, trackElements.album)
			if err != nil {
				slog.Warn("failed to get release year from MusicBrainz for file", "error", err)
			}
			trackElements.year = releaseYear
			slog.Debug("track elements", "musicFileName", musicFileName, "trackElements", trackElements)
		}

		outDirPath := getOutDirPath(cfg.OutputDir, trackElements.artist, trackElements.album, trackElements.year)
		err = os.MkdirAll(outDirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outDirPath, err)
		}

		outfileName := musicFileName
		// Only set the outfileName if the track elements are not unknown
		if trackElements.artist != unknownArtist && trackElements.album != unknownAlbum && trackElements.trackNumber != unknownTrackNumber && trackElements.title != unknownTitle {
			outfileName = getOutFileName(trackElements.artist, trackElements.album, trackElements.trackNumber, trackElements.title, filepath.Ext(musicFileName))
		}
		slog.Debug("file name", "musicFileName", musicFileName, "name", outfileName)

		// Copy file to output directory
		outputFile := filepath.Join(outDirPath, outfileName)
		slog.Debug("copying file", "path", outputFile)
		if cfg.Copy {
			err = copyFile(musicFilePath, outputFile)
			if err != nil {
				return fmt.Errorf("failed to move file to output directory: %w", err)
			}
		} else {
			err = moveFile(musicFilePath, outputFile)
			if err != nil {
				return fmt.Errorf("failed to copy file to output directory: %w", err)
			}
		}
	}
	slog.Info("processed all files", "count", len(musicFilePaths))

	return nil
}

func sanitizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.ReplaceAll(tag, "/", "_")
	return tag
}

func getTrackElementsFromTags(tags map[string][]string) (trackElements, error) {
	var te trackElements

	artists, ok := tags["ARTIST"]
	if ok {
		te.artist = sanitizeTag(artists[0])
	} else {
		te.artist = unknownArtist
	}
	albums, ok := tags["ALBUM"]
	if ok {
		te.album = sanitizeTag(albums[0])
	} else {
		te.album = unknownAlbum
	}
	titles, ok := tags["TITLE"]
	if ok {
		te.title = sanitizeTag(titles[0])
	} else {
		te.title = unknownTitle
	}
	trackNumbers, ok := tags["TRACKNUMBER"]
	if ok {
		trackNumber := sanitizeTag(trackNumbers[0])
		re := regexp.MustCompile(`\d+`)
		trackNumber = re.FindString(trackNumber)
		te.trackNumber = trackNumber
	} else {
		te.trackNumber = unknownTrackNumber
	}
	date, ok := tags["DATE"]
	if ok {
		formattedDate, err := time.Parse("2006-01-02", date[0])
		if err != nil {
			return te, fmt.Errorf("failed to parse date %s: %w", date[0], err)
		}
		te.year = strconv.Itoa(formattedDate.Year())
	} else {
		te.year = unknownYear
	}

	return te, nil
}

func getOutDirPath(outDir, artist, album, year string) string {
	yearString := ""
	if year != unknownYear {
		yearString = fmt.Sprintf(" (%s)", year)
	}
	return filepath.Join(outDir, artist, fmt.Sprintf("%s%s", album, yearString))
}

func getOutFileName(artist, album, trackNumber, title, ext string) string {
	return fmt.Sprintf("%s - %s - %s %s%s", artist, album, trackNumber, title, ext)
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", src, err)
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", dst, err)
	}
	return nil
}

func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		if strings.Contains(err.Error(), "invalid cross-device link") {
			return moveCrossDevice(src, dst)
		}
		return fmt.Errorf("failed to move file %s to %s: %w", src, dst, err)
	}
	return nil
}

func moveCrossDevice(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", source, err)
	}
	dst, err := os.Create(destination)
	if err != nil {
		err := src.Close()
		if err != nil {
			return fmt.Errorf("failed to close source file %s: %w", source, err)
		}
		return fmt.Errorf("failed to create destination file %s: %w", destination, err)
	}
	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", source, destination, err)
	}
	err = src.Close()
	if err != nil {
		return fmt.Errorf("failed to close source file %s: %w", source, err)
	}
	err = dst.Close()
	if err != nil {
		return fmt.Errorf("failed to close destination file %s: %w", destination, err)
	}
	fi, err := os.Stat(source)
	if err != nil {
		err = os.Remove(destination)
		if err != nil {
			return fmt.Errorf("failed to remove destination file %s: %w", destination, err)
		}
		return fmt.Errorf("failed to stat destination file %s: %w", destination, err)
	}
	err = os.Chmod(destination, fi.Mode())
	if err != nil {
		err = os.Remove(destination)
		if err != nil {
			return fmt.Errorf("failed to remove destination file %s: %w", destination, err)
		}
		return fmt.Errorf("failed to chmod destination file %s: %w", destination, err)
	}
	err = os.Remove(source)
	if err != nil {
		return fmt.Errorf("failed to remove source file %s: %w", source, err)
	}
	return nil
}

func getReleaseYearFromMusicBrainz(mb *gomusicbrainz.WS2Client, artist, album string) (string, error) {
	rgQuery := fmt.Sprintf(`artist:"%s" AND releasegroup:"%s"`, artist, album)
	rgResp, err := mb.SearchReleaseGroup(rgQuery, -1, -1)
	if err != nil {
		return "", fmt.Errorf("failed to search MusicBrainz: %w", err)
	}
	if len(rgResp.ReleaseGroups) == 0 {
		slog.Warn("no release group found", "artist", artist, "album", album)
		return "", nil
	}
	return strconv.Itoa(rgResp.ReleaseGroups[0].FirstReleaseDate.Year()), nil
}

// func getTitleFromMusicBrainz(mb *gomusicbrainz.WS2Client, fileName string) (string, error) {
// 	queryString := sanitizeFileNameForMusicBrainzQuery(fileName)
// 	query := fmt.Sprintf(`"%s"`, queryString)
// 	resp, err := mb.SearchRecording(query, -1, -1)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to search MusicBrainz: %w", err)
// 	}
// 	if resp.Count == 0 {
// 		slog.Warn("no recording found", "fileName", queryString)
// 		return "", nil
// 	}

// 	return resp.Recordings[0].Title, nil
// }

// func getArtistFromMusicBrainz(mb *gomusicbrainz.WS2Client, fileName string) (string, error) {
// 	queryString := sanitizeFileNameForMusicBrainzQuery(fileName)
// 	query := fmt.Sprintf(`"%s"`, queryString)
// 	resp, err := mb.SearchArtist(query, -1, -1)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to search MusicBrainz: %w", err)
// 	}
// 	if resp.Count == 0 {
// 		slog.Warn("no artist found", "fileName", queryString)
// 		return "", nil
// 	}

// 	return resp.Artists[0].Name, nil
// }

// func sanitizeFileNameForMusicBrainzQuery(fileName string) string {
// 	// Remove all repeated spaces
// 	fileName = strings.ReplaceAll(fileName, "  ", " ")
// 	// Remove all non-alphanumeric characters
// 	return strings.Map(func(r rune) rune {
// 		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
// 			return r
// 		}
// 		return -1
// 	}, fileName)
// }
