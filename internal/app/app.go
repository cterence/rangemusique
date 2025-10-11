package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/michiwend/gomusicbrainz"
	"go.senan.xyz/taglib"
)

type TrackElements struct {
	Artist      string
	Album       string
	Title       string
	TrackNumber string
	Year        int
}

func Run(ctx context.Context, cfg Config) error {
	// List all files in the input directory recusively
	musicFiles := []string{}

	mb, err := gomusicbrainz.NewWS2Client("https://musicbrainz.org", "rangemusique", "1.0", "https://github.com/cterence")
	if err != nil {
		return fmt.Errorf("failed to create MusicBrainz client: %w", err)
	}

	err = filepath.WalkDir(cfg.InputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		musicFiles = append(musicFiles, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk input directory: %w", err)
	}

	for _, file := range musicFiles {
		tags, err := taglib.ReadTags(file)
		if err != nil {
			return fmt.Errorf("failed to read tags from file %s: %w", file, err)
		}

		trackElements, err := extractTrackElementsFromTags(tags)
		if err != nil {
			slog.Warn("failed to extract track elements from file %s: %v, skipping\n", file, err)
			continue
		}

		// Get year of the release
		releaseYear, err := getReleaseYearFromMusicBrainz(mb, trackElements.Artist, trackElements.Album)
		if err != nil {
			slog.Warn("failed to get release year from MusicBrainz for file", "error", err)
		}

		outputDirPath := getDirPath(cfg.OutputDir, trackElements.Artist, trackElements.Album, releaseYear)
		err = os.MkdirAll(outputDirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputDirPath, err)
		}

		fileName := getFileName(trackElements.Artist, trackElements.Album, trackElements.TrackNumber, trackElements.Title, filepath.Ext(file))

		// Copy file to output directory
		outputFile := filepath.Join(outputDirPath, fileName)
		slog.Debug("Copying file to %s\n", outputFile)
		if cfg.Copy {
			err = copyFile(file, outputFile)
			if err != nil {
				return fmt.Errorf("failed to move file to output directory: %w", err)
			}
		} else {
			err = moveFile(file, outputFile)
			if err != nil {
				return fmt.Errorf("failed to copy file to output directory: %w", err)
			}
		}
	}

	return nil
}

func extractTrackElementsFromTags(tags map[string][]string) (TrackElements, error) {
	var te TrackElements

	artists, ok := tags["ARTIST"]
	if !ok || len(artists) == 0 {
		return te, fmt.Errorf("no artist tag found")
	}
	albums, ok := tags["ALBUM"]
	if !ok || len(albums) == 0 {
		return te, fmt.Errorf("no album tag found")
	}
	titles, ok := tags["TITLE"]
	if !ok || len(titles) == 0 {
		return te, fmt.Errorf("no title tag found")
	}
	trackNumbers, ok := tags["TRACKNUMBER"]
	if !ok || len(trackNumbers) == 0 {
		return te, fmt.Errorf("no track number tag found")
	}

	te.Artist = artists[0]
	te.Album = albums[0]
	te.Title = titles[0]
	te.TrackNumber = trackNumbers[0]

	return te, nil
}

func getDirPath(outDir, artist, album string, year int) string {
	yearString := ""
	if year > 0 {
		yearString = fmt.Sprintf(" (%d)", year)
	}
	return filepath.Join(outDir, artist, fmt.Sprintf("%s%s", album, yearString))
}

func getFileName(artist, album, trackNumber, title, ext string) string {
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
	return os.Rename(src, dst)
}

func getReleaseYearFromMusicBrainz(mb *gomusicbrainz.WS2Client, artist, album string) (int, error) {
	rgQuery := fmt.Sprintf(`artist:"%s" AND releasegroup:"%s"`, artist, album)
	rgResp, err := mb.SearchReleaseGroup(rgQuery, -1, -1)
	if err != nil {
		return 0, fmt.Errorf("failed to search MusicBrainz: %w", err)
	}
	if len(rgResp.ReleaseGroups) == 0 {
		slog.Warn("no release group found", "artist", artist, "album", album)
		return 0, nil
	}
	return rgResp.ReleaseGroups[0].FirstReleaseDate.Time.Year(), nil
}
