package domain

import "testing"

func TestOrder_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name    string
		current OrderStatus
		next    OrderStatus
		want    bool
	}{
		{"accepted to cooking", OrderStatusAccepted, OrderStatusCooking, true},
		{"accepted to cancelled_by_user", OrderStatusAccepted, OrderStatusCancelledByUser, true},
		{"cooking to ready", OrderStatusCooking, OrderStatusReady, true},
		{"cooking to cancelled_by_venue", OrderStatusCooking, OrderStatusCancelledByVenue, true},
		{"cooking to cancelled_by_user is invalid, too late", OrderStatusCooking, OrderStatusCancelledByUser, false},
		{"ready to cooking is invalid, no going back", OrderStatusReady, OrderStatusCooking, false},
		{"rejected has no transitions", OrderStatusRejected, OrderStatusCooking, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := Order{Status: tt.current}
			if got := o.CanTransitionTo(tt.next); got != tt.want {
				t.Errorf("CanTransitionTo(%q) with status %q = %v, want %v", tt.next, tt.current, got, tt.want)
			}
		})
	}
}
