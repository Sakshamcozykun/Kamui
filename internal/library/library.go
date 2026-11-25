package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	Title string    `json:"title"`
	Added time.Time `json:"added"`
}
type Store struct{ path string }

func New() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	dir = filepath.Join(dir, "kamui")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(dir, "library.json")}, nil
}
func (s *Store) load() (map[string][]Entry, error) {
	data := map[string][]Entry{}
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return data, nil
	}
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &data)
	return data, err
}
func (s *Store) save(data map[string][]Entry) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}
func (s *Store) Add(list, title string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	entries := data[list][:0]
	for _, e := range data[list] {
		if strings.EqualFold(e.Title, title) {
			if list == "watchlist" {
				return nil
			}
			continue
		}
		entries = append(entries, e)
	}
	data[list] = append(entries, Entry{Title: title, Added: time.Now().UTC()})
	return s.save(data)
}
func (s *Store) Remove(list, title string) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	out := data[list][:0]
	for _, e := range data[list] {
		if !strings.EqualFold(e.Title, title) {
			out = append(out, e)
		}
	}
	data[list] = out
	return s.save(data)
}
func (s *Store) List(list string) ([]Entry, error) { d, err := s.load(); return d[list], err }
