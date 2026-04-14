package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/example/ai-integration-test-demo/internal/breakpoint"
	"github.com/example/ai-integration-test-demo/internal/equipment"
	"github.com/example/ai-integration-test-demo/internal/event"
	"github.com/example/ai-integration-test-demo/internal/player"
	"github.com/gorilla/websocket"
)

type Request struct {
	Cmd      string `json:"cmd"`
	PlayerID int    `json:"playerId,omitempty"`
	Sub      string `json:"sub,omitempty"`
	ItemID   int    `json:"itemId,omitempty"`
	Count    int    `json:"count,omitempty"`
	TaskID   int    `json:"taskId,omitempty"`
	AchID    int    `json:"achId,omitempty"`
	Day      int    `json:"day,omitempty"`
	MailID   int    `json:"mailId,omitempty"`
	Slot     string `json:"slot,omitempty"`
}

type Response struct {
	Ok   bool     `json:"ok"`
	Data any      `json:"data,omitempty"`
	Log  []string `json:"log,omitempty"`
	Err  string   `json:"err,omitempty"`
}

type Server struct {
	pm       *player.Manager
	bus      *event.Bus
	bp       *breakpoint.Controller
	upgrader websocket.Upgrader
}

func New(pm *player.Manager, bus *event.Bus, bp *breakpoint.Controller) *Server {
	return &Server{
		pm:  pm,
		bus: bus,
		bp:  bp,
		upgrader: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
			return true
		}},
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("client connected: %s", conn.RemoteAddr())

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		var req Request
		if err := json.Unmarshal(msg, &req); err != nil {
			s.send(conn, Response{Ok: false, Err: "invalid json"})
			continue
		}

		resp := s.dispatch(req)
		s.send(conn, resp)
	}
}

func (s *Server) send(conn *websocket.Conn, resp Response) {
	data, _ := json.Marshal(resp)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("write error: %v", err)
	}
}

func (s *Server) dispatch(req Request) Response {
	switch req.Cmd {
	case "playermgr":
		return s.handlePlayerMgr(req)
	case "additem":
		return s.handleAddItem(req)
	case "removeitem":
		return s.handleRemoveItem(req)
	case "checkin":
		return s.handleCheckIn(req)
	case "claimreward":
		return s.handleClaimReward(req)
	case "equip":
		return s.handleEquip(req)
	case "unequip":
		return s.handleUnequip(req)
	case "claimmail":
		return s.handleClaimMail(req)
	case "next":
		return s.handleNext(req)
	case "batch":
		return s.handleBatch()
	case "help":
		return s.handleHelp()
	default:
		return Response{Ok: false, Err: fmt.Sprintf("unknown cmd: %s", req.Cmd)}
	}
}

func (s *Server) handlePlayerMgr(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}

	switch req.Sub {
	case "bag":
		if req.ItemID != 0 {
			it := p.Bag.GetItem(req.ItemID)
			if it == nil {
				return Response{Ok: true, Data: nil}
			}
			return Response{Ok: true, Data: it}
		}
		return Response{Ok: true, Data: p.Bag.AllItems()}
	case "task":
		if req.TaskID != 0 {
			t := p.Tasks.GetTask(req.TaskID)
			if t == nil {
				return Response{Ok: true, Data: nil}
			}
			return Response{Ok: true, Data: t}
		}
		return Response{Ok: true, Data: p.Tasks.AllTasks()}
	case "achievement":
		if req.AchID != 0 {
			a := p.Achievements.GetAchievement(req.AchID)
			if a == nil {
				return Response{Ok: true, Data: nil}
			}
			return Response{Ok: true, Data: a}
		}
		return Response{Ok: true, Data: p.Achievements.AllAchievements()}
	case "equipment":
		return Response{Ok: true, Data: p.Equipment.All()}
	case "signin":
		if req.Day != 0 {
			return Response{Ok: true, Data: p.SignIn.GetDay(req.Day)}
		}
		return Response{Ok: true, Data: p.SignIn.AllDays()}
	case "mail":
		if req.MailID != 0 {
			return Response{Ok: true, Data: p.Mail.GetMail(req.MailID)}
		}
		return Response{Ok: true, Data: p.Mail.AllMails()}
	default:
		return Response{
			Ok: true,
			Data: map[string]any{
				"playerId": p.ID,
				"modules":  []string{"bag", "task", "achievement", "equipment", "signin", "mail"},
			},
		}
	}
}

func (s *Server) handleAddItem(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.Bag.AddItem(req.ItemID, req.Count)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleRemoveItem(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.Bag.RemoveItem(req.ItemID, req.Count)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleCheckIn(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.SignIn.CheckIn(req.Day)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleClaimReward(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.SignIn.ClaimReward(req.Day)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleEquip(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.Equipment.Equip(equipment.EquipSlot(req.Slot), req.ItemID)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleUnequip(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.Equipment.Unequip(equipment.EquipSlot(req.Slot))
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleClaimMail(req Request) Response {
	p := s.pm.GetPlayer(req.PlayerID)
	if p == nil {
		return Response{Ok: false, Err: "player not found"}
	}
	s.bp.Enqueue(breakpoint.PendingOp{
		Execute: func() {
			p.Mail.ClaimAttachment(req.MailID)
		},
	})
	return Response{Ok: true, Data: map[string]any{"queued": true, "pendingOps": s.bp.PendingCount()}}
}

func (s *Server) handleNext(req Request) Response {
	logs := s.bp.Next()
	return Response{Ok: true, Log: logs}
}

func (s *Server) handleBatch() Response {
	var allLogs []string
	for {
		logs := s.bp.Next()
		if len(logs) == 0 {
			break
		}
		allLogs = append(allLogs, logs...)
	}
	return Response{Ok: true, Log: allLogs}
}

func (s *Server) handleHelp() Response {
	return Response{
		Ok: true,
		Data: map[string]any{
			"commands": []map[string]string{
				{"cmd": "playermgr", "desc": "query player data, sub: bag/task/achievement/equipment/signin/mail"},
				{"cmd": "additem", "desc": "enqueue add item (requires playerId, itemId, count)"},
				{"cmd": "removeitem", "desc": "enqueue remove item (requires playerId, itemId, count)"},
				{"cmd": "checkin", "desc": "enqueue sign-in check-in (requires playerId, day)"},
				{"cmd": "claimreward", "desc": "enqueue claim sign-in reward (requires playerId, day)"},
				{"cmd": "equip", "desc": "enqueue equip item (requires playerId, slot, itemId)"},
				{"cmd": "unequip", "desc": "enqueue unequip slot (requires playerId, slot)"},
				{"cmd": "claimmail", "desc": "enqueue claim mail attachment (requires playerId, mailId)"},
				{"cmd": "next", "desc": "execute one pending operation and return logs"},
				{"cmd": "batch", "desc": "execute all pending operations and return all logs"},
				{"cmd": "help", "desc": "show this help"},
			},
		},
	}
}
