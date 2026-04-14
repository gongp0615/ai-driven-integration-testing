package mail

import (
	"fmt"

	"github.com/example/ai-integration-test-demo/internal/event"
)

type Mail struct {
	MailID     int         `json:"mailId"`
	Subject    string      `json:"subject"`
	Attachment *Attachment `json:"attachment,omitempty"`
	Claimed    bool        `json:"claimed"`
}

type Attachment struct {
	ItemID int `json:"itemId"`
	Count  int `json:"count"`
}

type MailSystem struct {
	playerID int
	mails    map[int]*Mail
	nextID   int
	bus      *event.Bus
}

func New(playerID int, bus *event.Bus) *MailSystem {
	ms := &MailSystem{
		playerID: playerID,
		mails:    make(map[int]*Mail),
		nextID:   1,
		bus:      bus,
	}
	bus.Subscribe("achievement.unlocked", ms.onAchievementUnlocked)
	bus.Subscribe("signin.claimed", ms.onSignInClaimed)
	return ms
}

func (ms *MailSystem) onAchievementUnlocked(e event.Event) {
	playerID, _ := e.Data["playerID"].(int)
	if playerID != ms.playerID {
		return
	}
	achID, _ := e.Data["achID"].(int)
	ms.SendMail(fmt.Sprintf("Achievement Unlocked! (id=%d)", achID), nil)
}

func (ms *MailSystem) onSignInClaimed(e event.Event) {
	playerID, _ := e.Data["playerID"].(int)
	if playerID != ms.playerID {
		return
	}
	day, _ := e.Data["day"].(int)
	rewardItem, _ := e.Data["rewardItem"].(int)
	rewardCount, _ := e.Data["rewardCount"].(int)
	ms.SendMail(fmt.Sprintf("Sign-in Day %d Reward", day), &Attachment{
		ItemID: rewardItem,
		Count:  rewardCount,
	})
}

func (ms *MailSystem) SendMail(subject string, attachment *Attachment) {
	mailID := ms.nextID
	ms.nextID++
	ms.mails[mailID] = &Mail{
		MailID:     mailID,
		Subject:    subject,
		Attachment: attachment,
		Claimed:    false,
	}
	ms.bus.AppendLog(fmt.Sprintf("[Mail] sent: %s (id=%d)", subject, mailID))
	ms.bus.Publish(event.Event{
		Type: "mail.sent",
		Data: map[string]any{"playerID": ms.playerID, "mailId": mailID, "subject": subject},
	})
}

func (ms *MailSystem) ClaimAttachment(mailID int) {
	m, ok := ms.mails[mailID]
	if !ok {
		ms.bus.AppendLog(fmt.Sprintf("[Mail] claim failed: mail %d not found", mailID))
		return
	}
	if m.Claimed {
		ms.bus.AppendLog(fmt.Sprintf("[Mail] claim failed: mail %d already claimed", mailID))
		return
	}
	if m.Attachment == nil {
		ms.bus.AppendLog(fmt.Sprintf("[Mail] claim failed: mail %d has no attachment", mailID))
		return
	}
	m.Claimed = true
	ms.bus.AppendLog(fmt.Sprintf("[Mail] claimed attachment from mail %d: item %d x%d", mailID, m.Attachment.ItemID, m.Attachment.Count))
	ms.bus.Publish(event.Event{
		Type: "mail.claimed",
		Data: map[string]any{
			"playerID": ms.playerID,
			"mailId":   mailID,
			"itemID":   m.Attachment.ItemID,
			"count":    m.Attachment.Count,
		},
	})
}

func (ms *MailSystem) GetMail(mailID int) *Mail {
	return ms.mails[mailID]
}

func (ms *MailSystem) AllMails() []*Mail {
	out := make([]*Mail, 0, len(ms.mails))
	for _, m := range ms.mails {
		out = append(out, m)
	}
	return out
}
