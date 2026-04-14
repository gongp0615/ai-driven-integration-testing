package signin

import (
	"fmt"

	"github.com/example/ai-integration-test-demo/internal/event"
)

type SignInDay struct {
	Day         int  `json:"day"`
	RewardItem  int  `json:"rewardItem"`
	RewardCount int  `json:"rewardCount"`
	Claimed     bool `json:"claimed"`
}

type SignInSystem struct {
	playerID int
	days     map[int]*SignInDay
	bus      *event.Bus
}

var defaultRewards = map[int][2]int{
	1: {2001, 1},
	2: {2002, 1},
	3: {2001, 2},
	4: {2002, 2},
	5: {2001, 3},
	6: {2002, 3},
	7: {3001, 1},
}

func New(playerID int, bus *event.Bus) *SignInSystem {
	ss := &SignInSystem{
		playerID: playerID,
		days:     make(map[int]*SignInDay),
		bus:      bus,
	}
	for day, reward := range defaultRewards {
		ss.days[day] = &SignInDay{
			Day:         day,
			RewardItem:  reward[0],
			RewardCount: reward[1],
			Claimed:     false,
		}
	}
	return ss
}

func (ss *SignInSystem) CheckIn(day int) {
	d, ok := ss.days[day]
	if !ok {
		ss.bus.AppendLog(fmt.Sprintf("[SignIn] invalid day %d", day))
		return
	}
	if d.Claimed {
		ss.bus.AppendLog(fmt.Sprintf("[SignIn] day %d already claimed", day))
		return
	}
	d.Claimed = true
	ss.bus.AppendLog(fmt.Sprintf("[SignIn] day %d claimed, reward: item %d x%d", day, d.RewardItem, d.RewardCount))
	ss.bus.Publish(event.Event{
		Type: "signin.claimed",
		Data: map[string]any{
			"playerID":    ss.playerID,
			"day":         day,
			"rewardItem":  d.RewardItem,
			"rewardCount": d.RewardCount,
		},
	})
}

func (ss *SignInSystem) ClaimReward(day int) {
	d, ok := ss.days[day]
	if !ok {
		ss.bus.AppendLog(fmt.Sprintf("[SignIn] invalid day %d", day))
		return
	}
	// Bug #3: no hasClaimedToday check — reward can be claimed repeatedly
	ss.bus.AppendLog(fmt.Sprintf("[SignIn] day %d reward claimed again", day))
	ss.bus.Publish(event.Event{
		Type: "signin.reward",
		Data: map[string]any{
			"playerID":    ss.playerID,
			"day":         day,
			"rewardItem":  d.RewardItem,
			"rewardCount": d.RewardCount,
		},
	})
}

func (ss *SignInSystem) GetDay(day int) *SignInDay {
	return ss.days[day]
}

func (ss *SignInSystem) AllDays() []*SignInDay {
	out := make([]*SignInDay, 0, len(ss.days))
	for _, d := range ss.days {
		out = append(out, d)
	}
	return out
}
