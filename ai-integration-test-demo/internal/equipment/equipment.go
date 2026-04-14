package equipment

import (
	"fmt"

	"github.com/example/ai-integration-test-demo/internal/event"
)

type EquipSlot string

const (
	SlotWeapon EquipSlot = "weapon"
	SlotArmor  EquipSlot = "armor"
)

type EquippedItem struct {
	Slot   EquipSlot `json:"slot"`
	ItemID int       `json:"itemId"`
}

type EquipmentSystem struct {
	playerID int
	slots    map[EquipSlot]int
	bus      *event.Bus
}

var equipableItems = map[int]EquipSlot{
	3001: SlotWeapon,
	3002: SlotArmor,
}

func New(playerID int, bus *event.Bus) *EquipmentSystem {
	es := &EquipmentSystem{
		playerID: playerID,
		slots:    make(map[EquipSlot]int),
		bus:      bus,
	}
	bus.Subscribe("item.added", es.onItemAdded)
	return es
}

func (es *EquipmentSystem) onItemAdded(e event.Event) {
	playerID, _ := e.Data["playerID"].(int)
	if playerID != es.playerID {
		return
	}
	itemID, _ := e.Data["itemID"].(int)

	if slot, ok := equipableItems[itemID]; ok {
		es.Equip(slot, itemID)
	}
}

func (es *EquipmentSystem) Equip(slot EquipSlot, itemID int) {
	es.slots[slot] = itemID
	es.bus.AppendLog(fmt.Sprintf("[Equipment] auto-equip: %s slot → item %d", slot, itemID))
	es.bus.Publish(event.Event{
		Type: "equip.success",
		Data: map[string]any{"playerID": es.playerID, "slot": string(slot), "itemID": itemID},
	})
}

func (es *EquipmentSystem) Unequip(slot EquipSlot) {
	if itemID, ok := es.slots[slot]; ok {
		delete(es.slots, slot)
		es.bus.AppendLog(fmt.Sprintf("[Equipment] unequip: %s slot (was item %d)", slot, itemID))
		es.bus.Publish(event.Event{
			Type: "equip.unequipped",
			Data: map[string]any{"playerID": es.playerID, "slot": string(slot), "itemID": itemID},
		})
	}
}

func (es *EquipmentSystem) GetSlot(slot EquipSlot) *EquippedItem {
	if itemID, ok := es.slots[slot]; ok {
		return &EquippedItem{Slot: slot, ItemID: itemID}
	}
	return nil
}

func (es *EquipmentSystem) All() map[EquipSlot]*EquippedItem {
	out := make(map[EquipSlot]*EquippedItem)
	for slot, itemID := range es.slots {
		out[slot] = &EquippedItem{Slot: slot, ItemID: itemID}
	}
	return out
}

func (es *EquipmentSystem) HasWeapon() bool {
	_, ok := es.slots[SlotWeapon]
	return ok
}

func (es *EquipmentSystem) HasArmor() bool {
	_, ok := es.slots[SlotArmor]
	return ok
}
