package rest

import (
	"testing"

	controller "github.com/mdhasib01/go-rest-starter/controller"
)

// The permission ids are seeded by the migrations and must match.
func TestManageBloodBanksPermissionID(t *testing.T) {
	if ManageBloodBanksPermission != 10 || controller.ManageBloodBanksPermission != ManageBloodBanksPermission {
		t.Fatalf("ManageBloodBanksPermission must be 10 in rest and controller (migration 000003), got %d / %d",
			ManageBloodBanksPermission, controller.ManageBloodBanksPermission)
	}
}
