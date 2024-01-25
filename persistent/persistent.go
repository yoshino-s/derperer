package persistent

import (
	"encoding/json"
	"os"
	"sync"
)

type Persistent struct {
	RootPath string
	locks    map[string]*sync.RWMutex
}

func NewPersistent(rootPath string) (*Persistent, error) {
	if err := os.MkdirAll(rootPath, os.ModePerm); err != nil {
		return nil, err
	}

	return &Persistent{
		RootPath: rootPath,
		locks:    make(map[string]*sync.RWMutex),
	}, nil
}

func (p *Persistent) mutex(filename string) *sync.RWMutex {
	lock, ok := p.locks[filename]
	if !ok {
		lock = &sync.RWMutex{}
		p.locks[filename] = lock
	}

	return lock
}

func (p *Persistent) Save(filename string, data interface{}) error {
	lock := p.mutex(filename)
	lock.Lock()
	defer lock.Unlock()

	file, err := os.Create(p.RootPath + "/" + filename)
	if err != nil {
		return err
	}
	defer file.Close()

	d, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if _, err := file.Write(d); err != nil {
		return err
	}

	return nil
}

func (p *Persistent) Load(filename string, data interface{}) error {
	lock := p.mutex(filename)
	lock.RLock()
	defer lock.RUnlock()

	file, err := os.Open(p.RootPath + "/" + filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(data); err != nil {
		return err
	}

	return nil
}

func (p *Persistent) LoadOrCreate(filename string, data interface{}) error {
	lock := p.mutex(filename)
	lock.RLock()

	if _, err := os.Stat(p.RootPath + "/" + filename); os.IsNotExist(err) {
		lock.RUnlock()
		return p.Save(filename, data)
	} else {
		lock.RUnlock()
		return p.Load(filename, data)
	}
}
