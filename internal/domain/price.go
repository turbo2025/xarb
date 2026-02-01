package domain

import "strconv"

// Direction represents the price movement direction
type Direction int

const (
	DirectionSame Direction = 0
	DirectionUp   Direction = +1
	DirectionDown Direction = -1
)

// PriceState holds the state of a single price point
type PriceState struct {
	String    string
	Number    float64
	HasValue  bool
	Direction Direction
	IsSeen    bool
	IsParsed  bool
}

// Update updates the price state with a new price value
func (ps *PriceState) Update(price string) bool {
	if price == ps.String {
		ps.IsSeen = true
		return false
	}

	oldString := ps.String
	ps.String = price
	ps.IsSeen = true

	n, err := strconv.ParseFloat(price, 64)
	if err != nil {
		ps.IsParsed = false
		ps.Direction = DirectionSame
		return true
	}
	ps.IsParsed = true

	if !ps.HasValue {
		ps.HasValue = true
		ps.Number = n
		ps.Direction = DirectionSame
		return oldString != "" // Return true only if we had a previous value
	}

	prev := ps.Number
	switch {
	case n > prev:
		ps.Direction = DirectionUp
	case n < prev:
		ps.Direction = DirectionDown
	default:
		ps.Direction = DirectionSame
	}
	ps.Number = n
	return true
}
