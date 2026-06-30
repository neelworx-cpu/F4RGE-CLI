package managedlogin

import (
	"fmt"
	"time"

	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/controlplane"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/modelcatalog"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/promptbundle"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/runtimebundle"
	f4rgesession "github.com/neelworx-cpu/F4RGE-CLI/internal/f4rge/session"
)

func Finalize(controlPlane controlplane.Client, poll *controlplane.CLIAuthPollResponse) error {
	if poll == nil || poll.RuntimeSession.Token == "" {
		return fmt.Errorf("sign-in did not return a runtime session")
	}
	organizationID := poll.RuntimeSession.OrganizationID
	if organizationID == "" {
		organizationID = poll.OrganizationID
	}
	if organizationID == "" {
		return fmt.Errorf("sign-in completed but no organization was returned")
	}
	session := f4rgesession.ManagedSession{
		AccessToken:      poll.RuntimeSession.Token,
		RuntimeSessionID: poll.RuntimeSession.SessionID,
		RuntimeScopes:    poll.RuntimeSession.Scopes,
		SubjectUserID:    poll.RuntimeSession.SubjectUserID,
		UserDisplayName:  poll.RuntimeSession.DisplayName,
		UserEmail:        poll.RuntimeSession.Email,
		OrganizationID:   organizationID,
		OrganizationName: poll.RuntimeSession.OrganizationName,
		GatewayEndpoint:  controlPlane.BaseURL,
		PlatformEndpoint: controlPlane.BaseURL,
	}
	if expiresAt, parseErr := time.Parse(time.RFC3339, poll.RuntimeSession.ExpiresAt); parseErr == nil {
		session.ExpiresAt = expiresAt.Unix()
	}
	runtime, err := runtimebundle.Fetch(&session)
	if err != nil {
		return err
	}
	models, err := modelcatalog.Fetch(&session)
	if err != nil {
		return err
	}
	prompts, err := promptbundle.Fetch(&session)
	if err != nil {
		return err
	}

	session.OrganizationID = runtime.OrganizationID
	session.ModelCatalogVersion = models.CatalogVersion
	session.PromptBundleVersion = prompts.Snapshot.SnapshotID
	session.RuntimeBundleHash = runtime.ModelCatalog.CanonicalHash + ":" + runtime.PromptBundle.CanonicalHash + ":" + runtime.RuntimeConfig.Version
	if runtime.Session != nil {
		session.SubjectUserID = runtime.Session.SubjectUserID
		if session.UserDisplayName == "" {
			session.UserDisplayName = runtime.Session.DisplayName
		}
		if session.UserEmail == "" {
			session.UserEmail = runtime.Session.Email
		}
		if session.OrganizationName == "" {
			session.OrganizationName = runtime.Session.OrganizationName
		}
	}
	if session.SubjectUserID == "" {
		session.SubjectUserID = poll.RuntimeSession.SubjectUserID
	}
	if session.UserEmail == "" {
		session.UserEmail = "unavailable"
	}
	if session.UserDisplayName == "" {
		session.UserDisplayName = session.UserEmail
	}
	if session.OrganizationName == "" {
		session.OrganizationName = runtime.OrganizationID
	}
	if err := runtimebundle.Validate(runtime, &session); err != nil {
		return err
	}
	if err := modelcatalog.Validate(models, &session); err != nil {
		return err
	}
	if err := promptbundle.Validate(prompts, &session); err != nil {
		return err
	}
	if !f4rgesession.IsUsable(&session) {
		return fmt.Errorf("sign-in incomplete: missing %v", f4rgesession.MissingReadinessFields(&session))
	}
	return f4rgesession.Save(session)
}
