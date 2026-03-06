package web

import (
	"net/http"

	"neuralclaw/internal/security"
)

type securitySummary struct {
	PendingApprovals   int
	QuarantinedItems   int
	RecentBlockedTasks int
	RecentEvents       []security.SecurityEvent
}

func (s *Server) buildSecuritySummary(scope string) securitySummary {
	if s.guard == nil {
		return securitySummary{}
	}
	summary := securitySummary{}
	approvals, _ := s.guard.ListApprovals(scope)
	for _, approval := range approvals {
		if approval.Status == security.ApprovalPending {
			summary.PendingApprovals++
		}
	}
	quarantine, _ := s.guard.ListQuarantine(scope)
	summary.QuarantinedItems = len(quarantine)
	events, _ := s.guard.ListEvents(scope, 20)
	summary.RecentEvents = events
	for _, event := range events {
		if event.EventType == security.EventPromptBlocked {
			summary.RecentBlockedTasks++
		}
	}
	return summary
}

func (s *Server) handleSecurityOverview(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	data := struct {
		Scope   string
		Summary securitySummary
	}{
		Scope:   scope,
		Summary: s.buildSecuritySummary(scope),
	}
	s.renderTemplate(w, "security.html", data)
}

func (s *Server) handleSecurityApprovals(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	var approvals []security.ApprovalRequest
	if s.guard != nil {
		approvals, _ = s.guard.ListApprovals(scope)
	}
	data := struct {
		Scope     string
		Approvals []security.ApprovalRequest
	}{
		Scope:     scope,
		Approvals: approvals,
	}
	s.renderTemplate(w, "security_approvals.html", data)
}

func (s *Server) handleSecurityQuarantine(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	var items []security.QuarantineRecord
	if s.guard != nil {
		items, _ = s.guard.ListQuarantine(scope)
	}
	data := struct {
		Scope string
		Items []security.QuarantineRecord
	}{
		Scope: scope,
		Items: items,
	}
	s.renderTemplate(w, "security_quarantine.html", data)
}

func (s *Server) handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	var events []security.SecurityEvent
	if s.guard != nil {
		events, _ = s.guard.ListEvents(scope, 100)
	}
	data := struct {
		Scope  string
		Events []security.SecurityEvent
	}{
		Scope:  scope,
		Events: events,
	}
	s.renderTemplate(w, "security_events.html", data)
}
