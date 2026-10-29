package names

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	url            = "https://market.csgo.com/api/v2/dictionary/names.json"
	reloadInterval = 60 //min
)

type dictResp struct {
	Success bool   `json:"success"`
	Time    int64  `json:"time"`
	Items   []item `json:"items"`
}

type item struct {
	ID       int    `json:"id,string"`
	HashName string `json:"hash_name"`
}

type Dict struct {
	names atomic.Pointer[map[int]string]
}

func (d *Dict) Name(id int) (string, bool) {
	m := d.names.Load()
	if m == nil {
		return "", false
	}
	name, ok := (*m)[id]
	return name, ok
}

func (d *Dict) Len() int {
	m := d.names.Load()
	if m == nil {
		return 0
	}
	return len(*m)
}

func (d *Dict) Load(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rc, err := fetch(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	m, err := parse(rc)
	if err != nil {
		return err
	}
	lenOldM := d.Len()
	lenNewM := len(m)
	result := shouldReplace(lenOldM, lenNewM)
	if !result {
		return fmt.Errorf("new dictionary invalid: %d", lenNewM)
	}
	d.names.Store(&m)
	log.Printf("success load dictionary %d", d.Len())
	return nil
}

func (d *Dict) Run(ctx context.Context) error {
	ticker := time.NewTicker(reloadInterval * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		err := d.Load(ctx)
		if err != nil {
			log.Printf("dictionary run error, %v", err)
		}
	}
}

func fetch(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bad get dictionary: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("get dictionary status code: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func parse(r io.Reader) (map[int]string, error) {
	var dictResp dictResp
	if err := json.NewDecoder(r).Decode(&dictResp); err != nil {
		return nil, fmt.Errorf("decode dictionary: %w", err)
	}
	if !dictResp.Success {
		return nil, fmt.Errorf("market returned success=false")
	}
	var m = make(map[int]string, len(dictResp.Items))
	for _, v := range dictResp.Items {
		m[v.ID] = v.HashName
	}
	return m, nil
}

// old = 1000,  new = 300

func shouldReplace(oldLen, newLen int) bool {
	if newLen == 0 {
		return false
	}

	if oldLen == 0 {
		return true
	}

	if newLen <= (oldLen / 2) {
		return false
	}

	return true
}
