package domain

import "strings"

type Direction struct {
	From string
	To   string
}

func (direction Direction) Normalized() Direction {
	return Direction{
		From: strings.TrimSpace(direction.From),
		To:   strings.TrimSpace(direction.To),
	}
}
