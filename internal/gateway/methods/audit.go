package methods

import (
	"github.com/nextlevelbuilder/zalosclaw/internal/bus"
	"github.com/nextlevelbuilder/zalosclaw/internal/gateway"
	"github.com/nextlevelbuilder/zalosclaw/pkg/protocol"
)

// emitAudit broadcasts an audit event via eventBus for async persistence.
func emitAudit(pub bus.EventPublisher, client *gateway.Client, action, entityType, entityID string) {
	if pub == nil {
		return
	}
	pub.Broadcast(bus.Event{
		Name: protocol.EventAuditLog,
		Payload: bus.AuditEventPayload{
			ActorType:  "user",
			ActorID:    client.UserID(),
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			IPAddress:  client.RemoteAddr(),
		},
	})
}
