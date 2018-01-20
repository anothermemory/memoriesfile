package memoriesfile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/anothermemory/memories"
	"github.com/anothermemory/memory"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type file struct {
	path   string
	items  map[string]memory.Interface
	fs     afero.Fs
	fsUtil *afero.Afero
}

// New creates new instance which stores memories using given path
func New(path string) (memories.Interface, error) {
	return create(path, afero.NewOsFs())
}

// NewInMemory creates new instance which stores memories using memory file system
func NewInMemory() (memories.Interface, error) {
	return create("/anothermemory/memories.json", afero.NewMemMapFs())
}

func (f *file) Add(name string, m memory.Interface) error {
	f.items[name] = m
	return f.save()
}

func (f *file) Remove(name string) error {
	delete(f.items, name)
	return f.save()
}

func (f *file) Get(name string) (memory.Interface, error) {
	i, ok := f.items[name]
	if !ok {
		return nil, errors.Errorf("No memory available with name: %s", name)
	}

	return i, nil
}

func (f *file) GetAll() (map[string]memory.Interface, error) {
	return f.items, nil
}

func (f *file) RemoveAll() error {
	f.items = make(map[string]memory.Interface)
	return f.save()
}

func create(path string, fs afero.Fs) (memories.Interface, error) {
	s := &file{path: path, items: make(map[string]memory.Interface), fs: fs, fsUtil: &afero.Afero{Fs: fs}}
	err := s.load()
	if err != nil {
		return nil, err
	}

	return s, nil
}

type fileJSON struct {
	Items []json.RawMessage `json:"items"`
}

func (f *file) load() error {
	stat, err := f.fs.Stat(f.path)
	if os.IsNotExist(err) {
		return nil
	}

	if stat.IsDir() {
		return errors.New("Configured path is directory")
	}

	data, err := f.fsUtil.ReadFile(f.path)
	if err != nil {
		return errors.Wrap(err, "failed to read config file")
	}

	var j fileJSON
	err = json.Unmarshal(data, &j)

	if err != nil {
		return errors.Wrap(err, "failed to unmarshal config file")
	}

	for _, i := range j.Items {
		m, err := memory.NewFromJSONConfig(i)
		if nil != err {
			return errors.Wrap(err, "Failed to unmarshal memory")
		}
		f.items[m.Name()] = m
	}

	return nil
}

func (f *file) save() error {
	var data []byte
	var err error
	dir := filepath.Dir(f.path)
	err = f.fs.MkdirAll(dir, os.ModePerm)
	if nil != err {
		return errors.WithStack(err)
	}

	j := &fileJSON{}
	for _, s := range f.items {
		data, err = json.Marshal(s)
		if nil != err {
			return errors.Wrapf(err, "Failed to marshal memory: %s", s.Name())
		}
		j.Items = append(j.Items, data)
	}

	data, err = json.Marshal(j)

	if nil != err {
		return errors.Wrap(err, "Failed to serialize config file")
	}

	return errors.WithStack(f.fsUtil.WriteFile(f.path, data, os.ModePerm))
}
