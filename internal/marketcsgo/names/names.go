package names

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

const (
	url = "https://market.csgo.com/api/v2/dictionary/names.json"
)

type DictResp struct {
	Success bool   `json:"success"`
	Time    int64  `json:"time"`
	Items   []Item `json:"items"`
}

type Item struct {
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
	if !ok {
		return "", false
	}
	return name, true
}

func (d *Dict) Len() int {
	m := d.names.Load()
	if m == nil {
		return 0
	}
	return len(*m)
}

func (d *Dict) Run(ctx context.Context) error {
	rc, err := fetch(ctx)
	if err != nil {
		return err
	}
	m, err := parse(rc)
	if err != nil {
		return err
	}
	oldM := d.names.Load()
	lenOldM := len(*oldM)
	lenNewM := len(m)
	result := shouldReplace(lenOldM, lenNewM)
	if !result {
		return fmt.Errorf("new dictionary invalid: %d", lenNewM)
	}
	d.names.CompareAndSwap(oldM, &m)
	return nil
}

func fetch(ctx context.Context) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get dictionary: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get dictionary: %w", err)
	}
	defer resp.Body.Close()
	return resp.Body, nil
}

func parse(r io.Reader) (map[int]string, error) {
	var dictResp DictResp
	if err := json.NewDecoder(r).Decode(&dictResp); err != nil {
		return nil, fmt.Errorf("decode dictionary: %w", err)
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

	if oldLen == 0 && newLen != 0 {
		return true
	}

	if newLen <= (oldLen / 2) {
		return false
	}

	return true
}
