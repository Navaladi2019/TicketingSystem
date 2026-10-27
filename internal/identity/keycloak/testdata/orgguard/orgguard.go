// Package orgguard holds the call shapes TestPlaceInviteeIsTheOnlyRuntimeOrganizationsCaller
// must catch (or allow). It is type-checked by the test, never built: go
// ignores testdata directories.
package orgguard

import "context"

type account struct{}

type ClientCredentials struct{ ClientID string }

type admin struct{}

func (admin) getJSON(context.Context, *account, string, string, any, any) error { return nil }

func (admin) getWithCreds(context.Context, ClientCredentials, string) error { return nil }

type Directory struct {
	api        admin
	runtime    *account
	onboarding *account
}

const orgsPath = "/organizations"

// PlaceInvitee is the one allowed runtime caller.
func (d *Directory) PlaceInvitee(ctx context.Context, id string) error {
	return d.api.getJSON(ctx, d.runtime, "/organizations/members/{id}/organizations", "/organizations/members/"+id+"/organizations", nil, nil)
}

// FindOrg uses the onboarding account: allowed anywhere.
func (d *Directory) FindOrg(ctx context.Context) error {
	return d.api.getJSON(ctx, d.onboarding, "/organizations", "/organizations", nil, nil)
}

// Unrelated runtime calls are allowed.
func (d *Directory) Users(ctx context.Context) error {
	return d.api.getJSON(ctx, d.runtime, "/users", "/users", nil, nil)
}

// LiteralRuntime: a plain runtime call to /organizations outside PlaceInvitee.
func (d *Directory) LiteralRuntime(ctx context.Context) error {
	return d.api.getJSON(ctx, d.runtime, "/organizations", "/organizations", nil, nil)
}

// ConstPath: the path is a constant, so no string literal is in the call.
func (d *Directory) ConstPath(ctx context.Context) error {
	return d.api.getJSON(ctx, d.runtime, orgsPath, orgsPath, nil, nil)
}

// ConcatConst: a constant joined with a variable.
func (d *Directory) ConcatConst(ctx context.Context, id string) error {
	return d.api.getJSON(ctx, d.runtime, "/x", orgsPath+"/"+id, nil, nil)
}

// CopiedAccount: the runtime account copied to a local.
func (d *Directory) CopiedAccount(ctx context.Context) error {
	acct := d.runtime
	return d.api.getJSON(ctx, acct, "/organizations", "/organizations", nil, nil)
}

// CredsArg: raw credentials passed to an /organizations call.
func (d *Directory) CredsArg(ctx context.Context) error {
	return d.api.getWithCreds(ctx, ClientCredentials{ClientID: "x"}, "/organizations")
}
