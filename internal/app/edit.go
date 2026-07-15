package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	editcore "github.com/electrolux-oss/ik-tui/internal/edit"
	"github.com/electrolux-oss/ik-tui/internal/tabledata"
)

func (a *App) openEntityEditor(row tabledata.Row) {
	session, err := a.editSessionForRow(row)
	if err != nil {
		a.ui.OpenOverlayPrimitive("Edit", errorView(err.Error()))
		return
	}
	title := fmt.Sprintf("Edit %s", titleCase(session.Kind))
	if a.ui.OverlayVisible() {
		a.ui.CloseOverlay()
	}

	a.stopLiveLogStream()
	a.halt()

	var runErr error
	if ok := a.ui.Application().Suspend(func() {
		runErr = a.runEditSession(session)
	}); !ok {
		a.resume()
		a.ui.OpenOverlayPrimitive(title, errorView("Failed to start editor."))
		return
	}

	a.resume()

	if errors.Is(runErr, editcore.ErrEditCancelled) {
		a.requestRefresh()
		a.reopenEditedDetail(session)
		return
	}
	if runErr != nil {
		a.ui.OpenOverlayPrimitive(title, errorView(fmt.Sprintf("Edit failed for %s %s (%s).\n\n%v", session.Kind, valueOr(session.Name, session.ID), session.ID, runErr)))
		return
	}
	a.requestRefresh()
	if session.Kind == "resource" {
		a.openResourceReview(tabledata.Row{Raw: client.Resource{ID: session.ID, Name: session.Name}})
		return
	}
	a.reopenEditedDetail(session)
}

func (a *App) runEditSession(session editcore.Session) error {
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer loadCancel()

	if session.Load == nil || session.Apply == nil {
		return fmt.Errorf("edit is not supported for %s", session.Kind)
	}

	content, err := session.Load(loadCtx, session.ID)
	if err != nil {
		return err
	}

	path, err := editcore.PrepareSessionFile(session.Kind, session.Name, content)
	if err != nil {
		return err
	}
	defer os.Remove(path)

	before, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := editcore.OpenEditor(path, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return err
	}
	after, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Equal(before, after) {
		return editcore.ErrEditCancelled
	}

	applyCtx, applyCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer applyCancel()
	return session.Apply(applyCtx, session.ID, after)
}

func (a *App) editSessionForRow(row tabledata.Row) (editcore.Session, error) {
	var (
		session editcore.Session
		kind    string
		id      string
		name    string
	)
	switch value := row.Raw.(type) {
	case client.Resource:
		kind, id, name = "resource", value.ID, value.Name
	case client.Template:
		kind, id, name = "template", value.ID, value.Name
	case client.SourceCode:
		kind, id, name = "source_code", value.ID, valueOr(value.DisplayName(), value.ID)
	case client.Integration:
		kind, id, name = "integration", value.ID, value.Name
	case client.Storage:
		kind, id, name = "storage", value.ID, value.Name
	default:
		return session, fmt.Errorf("edit is not supported for %T", row.Raw)
	}
	descriptor, ok := a.registry.Resolve(kind)
	if !ok || descriptor.EditLoad == nil || descriptor.ApplyEdit == nil {
		return session, fmt.Errorf("edit is not supported for %s", kind)
	}
	session = editcore.Session{Kind: kind, ID: id, Name: name, Load: descriptor.EditLoad, Apply: descriptor.ApplyEdit}
	return session, nil
}

func (a *App) reopenEditedDetail(session editcore.Session) {
	switch session.Kind {
	case "resource":
		a.openResourceOverview(client.Resource{ID: session.ID, Name: session.Name})
	case "template":
		a.openTemplateOverview(session.ID, session.Name)
	case "source_code":
		a.openSourceCodeOverview(session.ID, session.Name)
	case "integration":
		a.openIntegrationOverview(session.ID, session.Name)
	case "storage":
		a.openStorageOverview(session.ID, session.Name)
	}
}
