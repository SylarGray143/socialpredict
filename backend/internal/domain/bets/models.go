package bets

import (
	"time"

	"socialpredict/internal/domain/boundary"
)

func newPlacedBoundaryBet(req PlaceRequest, outcome string, placedAt time.Time) *boundary.Bet {
	return &boundary.Bet{
		Username: req.Username,
		MarketID: req.MarketID,
		Amount:   req.Amount,
		Outcome:  outcome,
		PlacedAt: placedAt,
	}
}

// PlaceRequest captures the inputs required to place a buy bet.
type PlaceRequest struct {
	Username string
	MarketID uint
	Amount   int64
	Outcome  string
}

// NewBet builds the persisted bet record for a place request.
func (r PlaceRequest) NewBet(outcome string, placedAt time.Time) *boundary.Bet {
	return newPlacedBoundaryBet(r, outcome, placedAt)
}

// PlacedBet represents the bet that was successfully recorded.
type PlacedBet struct {
	Username string
	MarketID uint
	Amount   int64
	Outcome  string
	PlacedAt time.Time
}

func copyPlacedBet(target *PlacedBet, bet *boundary.Bet) *PlacedBet {
	if bet == nil {
		return nil
	}
	if target == nil {
		target = &PlacedBet{}
	}

	*target = PlacedBet{
		Username: bet.Username,
		MarketID: bet.MarketID,
		Amount:   bet.Amount,
		Outcome:  bet.Outcome,
		PlacedAt: bet.PlacedAt,
	}
	return target
}

// FromBoundary copies the persisted bet fields into the domain result shape.
func (p *PlacedBet) FromBoundary(bet *boundary.Bet) *PlacedBet {
	return copyPlacedBet(p, bet)
}

// FromModel preserves the legacy naming while reading from the boundary record.
func (p *PlacedBet) FromModel(bet *boundary.Bet) *PlacedBet {
	return p.FromBoundary(bet)
}
