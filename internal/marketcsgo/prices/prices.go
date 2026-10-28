package prices

type Item struct {
	MinPrice int
	Volume   int
}

type Table map[string]Item
