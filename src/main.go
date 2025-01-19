// This example demonstrates how to authenticate with Spotify using the authorization code flow.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//     - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/zmb3/spotify"
)

const lookbackInMonths = 13

func main() {
	tok := obtainOAuthToken()
	client := auth.NewClient(tok)

	// use the client to make calls that require authorization
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("You are logged in as", user.ID)

	playlists, err := getAllUserPlaylists(&client, user)
	if err != nil {
		log.Fatal(err)
	}

	likedSongs := getLikedSongs(&client)
	log.Printf("looking to %d liked songs.\n", len(likedSongs))
	monthlyPlaylists := divideSongsInMonthlyPlaylists(likedSongs)
	createOrCompleteThePlaylists(&client, playlists, monthlyPlaylists)
	log.Println("done.")
}

func createOrCompleteThePlaylists(client *spotify.Client, playlists []spotify.SimplePlaylist, monthlyPlaylists map[string][]spotify.SavedTrack) {
	for playlistName, songs := range monthlyPlaylists {
		playlist := getPlaylistFromName(playlists, playlistName)
		if playlist == nil {
			playlist = createPlaylist(client, playlistName)
		}
		addMissingSongsToPlaylist(client, playlist, songs)
	}
}

func getPlaylistFromName(playlists []spotify.SimplePlaylist, playlistName string) *spotify.SimplePlaylist {
	for _, playlist := range playlists {
		if playlist.Name == playlistName {
			return &playlist
		}
	}
	return nil
}

func createPlaylist(client *spotify.Client, playlistName string) *spotify.SimplePlaylist {
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}

	playlist, err := client.CreatePlaylistForUser(user.ID, playlistName, "", true)
	if err != nil {
		log.Fatal(err)
	}

	return &playlist.SimplePlaylist
}

func addMissingSongsToPlaylist(client *spotify.Client, playlist *spotify.SimplePlaylist, songs []spotify.SavedTrack) {
	var tracksIDs []spotify.ID

	playlistTracks := getPlaylistTracks(client, playlist)
	for _, song := range songs {
		if !isSongInPlaylist(song, playlistTracks) {
			tracksIDs = append(tracksIDs, song.ID)
		}
	}

	if len(tracksIDs) < 1 {
		return
	}

	_, err := client.AddTracksToPlaylist(playlist.ID, tracksIDs...)
	if err != nil {
		log.Fatal(err)
	}
}

func isSongInPlaylist(song spotify.SavedTrack, playlistTracks []spotify.PlaylistTrack) bool {
	for _, playlistSong := range playlistTracks {
		if song.ID == playlistSong.Track.ID {
			return true
		}
	}
	return false
}

func getPlaylistTracks(client *spotify.Client, playlist *spotify.SimplePlaylist) []spotify.PlaylistTrack {
	var tracks []spotify.PlaylistTrack
	var offset int
	var limit int
	offset = 0
	limit = 10

	opts := &spotify.Options{
		Offset: &offset,
		Limit:  &limit,
	}

	for {
		res, err := client.GetPlaylistTracksOpt(playlist.ID, opts, "")
		if err != nil {
			log.Fatal(err)
		}

		tracks = append(tracks, res.Tracks...)

		if res.Next == "" {
			break
		}

		offset = *opts.Offset + *opts.Limit
		opts.Offset = &offset
	}

	return tracks
}

func divideSongsInMonthlyPlaylists(songs []spotify.SavedTrack) map[string][]spotify.SavedTrack {
	monthlyPlaylists := make(map[string][]spotify.SavedTrack)

	for _, song := range songs {
		playlistName := playlistNameFromDateString(song.AddedAt)
		monthlyPlaylists[playlistName] = append(monthlyPlaylists[playlistName], song)
	}

	return monthlyPlaylists
}

func getLikedSongs(client *spotify.Client) []spotify.SavedTrack {
	var likedSongs []spotify.SavedTrack
	var offset int
	var limit int
	offset = 0
	limit = 50

	opts := &spotify.Options{
		Offset: &offset,
		Limit:  &limit,
	}

	log.Println("fetching liked songs. this may take a while...")
	for {
		res, err := client.CurrentUsersTracksOpt(opts)
		if err != nil {
			log.Fatal(err)
		}

		likedSongs = append(likedSongs, res.Tracks...)

		lastLikedSong := likedSongs[len(likedSongs)-1]
		if trackIsOlderThanNMonths(lastLikedSong, lookbackInMonths) {
			break
		}

		if res.Next == "" {
			break
		}

		offset = *opts.Offset + *opts.Limit
		opts.Offset = &offset
	}

	return likedSongs
}

func trackIsOlderThanNMonths(track spotify.SavedTrack, n int) bool {
	d, err := time.Parse("2006-01-02T15:04:05Z", track.AddedAt)
	if err != nil {
		log.Fatal(err)
	}

	return d.Before(time.Now().AddDate(0, -n, 0))
}

func getAllUserPlaylists(client *spotify.Client, user *spotify.PrivateUser) ([]spotify.SimplePlaylist, error) {
	var playlists []spotify.SimplePlaylist
	var offset int
	var limit int
	offset = 0
	limit = 10

	opts := &spotify.Options{
		Offset: &offset,
		Limit:  &limit,
	}

	for {
		res, err := client.GetPlaylistsForUserOpt(user.ID, opts)
		if err != nil {
			return nil, err
		}

		playlists = append(playlists, res.Playlists...)

		if res.Next == "" {
			break
		}

		offset = *opts.Offset + *opts.Limit
		opts.Offset = &offset
	}

	return playlists, nil
}

func playlistNameFromDateString(dateString string) string {
	d, err := time.Parse("2006-01-02T15:04:05Z", dateString)
	if err != nil {
		log.Fatal(err)
	}
	mon := d.Month().String()
	year := d.Year() % 1e2
	return fmt.Sprintf("%s '%d", mon, year)
}
