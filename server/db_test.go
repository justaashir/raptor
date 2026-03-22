package server

import (
	"raptor/model"
	"testing"
)

func TestDB_CreateAndGetTicket(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	ticket := model.NewTicket("Test ticket", "some content", "alice")
	err = db.CreateTicket(ticket)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	got, err := db.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if got.ID != ticket.ID {
		t.Fatalf("expected ID %q, got %q", ticket.ID, got.ID)
	}
	if got.Title != "Test ticket" {
		t.Fatalf("expected title %q, got %q", "Test ticket", got.Title)
	}
	if got.Content != "some content" {
		t.Fatalf("expected content %q, got %q", "some content", got.Content)
	}
	if got.Status != model.Todo {
		t.Fatalf("expected status %q, got %q", model.Todo, got.Status)
	}
	if got.CreatedBy != "alice" {
		t.Fatalf("expected created_by %q, got %q", "alice", got.CreatedBy)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDB_ListTickets_Empty(t *testing.T) {
	db := newTestDB(t)
	tickets, err := db.ListTickets("", "")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 0 {
		t.Fatalf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestDB_ListTickets_FilterByStatus(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Todo task", "", "")
	t2 := model.NewTicket("Done task", "", "")
	t2.Status = model.Done
	db.CreateTicket(t1)
	db.CreateTicket(t2)

	tickets, err := db.ListTickets("", "todo")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].Title != "Todo task" {
		t.Fatalf("expected %q, got %q", "Todo task", tickets[0].Title)
	}
}

func TestDB_ListTickets_NoStatusReturnsAll(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Task A", "", "")
	t2 := model.NewTicket("Task B", "", "")
	t2.Status = model.Done
	db.CreateTicket(t1)
	db.CreateTicket(t2)

	tickets, err := db.ListTickets("", "")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets (no closed filtering), got %d", len(tickets))
	}
}

func TestDB_UpdateTicket(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("Original", "", "")
	db.CreateTicket(ticket)

	err := db.UpdateTicket(ticket.ID, map[string]any{
		"title":  "Updated",
		"status": "in_progress",
	})
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	got, _ := db.GetTicket(ticket.ID)
	if got.Title != "Updated" {
		t.Fatalf("expected title %q, got %q", "Updated", got.Title)
	}
	if got.Status != model.InProgress {
		t.Fatalf("expected status %q, got %q", model.InProgress, got.Status)
	}
}

func TestDB_DeleteTicket(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("To delete", "", "")
	db.CreateTicket(ticket)

	err := db.DeleteTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = db.GetTicket(ticket.ID)
	if err == nil {
		t.Fatal("expected error getting deleted ticket")
	}
}

func TestDB_CreateWorkspace(t *testing.T) {
	db := newTestDB(t)
	err := db.CreateWorkspace("ws123456", "My Team", "alice")
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	workspaces, err := db.ListWorkspacesForUser("alice")
	if err != nil {
		t.Fatalf("failed to list workspaces: %v", err)
	}
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
	if workspaces[0].Name != "My Team" {
		t.Fatalf("expected name %q, got %q", "My Team", workspaces[0].Name)
	}
}

func TestDB_WorkspaceMembers(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")

	role, err := db.GetMemberRole("ws1", "alice")
	if err != nil {
		t.Fatalf("failed to get role: %v", err)
	}
	if role != "owner" {
		t.Fatalf("expected owner, got %q", role)
	}

	err = db.AddWorkspaceMember("ws1", "bob", "member")
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	members, err := db.ListWorkspaceMembers("ws1")
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	workspaces, _ := db.ListWorkspacesForUser("bob")
	if len(workspaces) != 1 {
		t.Fatalf("expected bob to see 1 workspace, got %d", len(workspaces))
	}
}

func TestDB_AddWorkspaceMember_Duplicate(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")

	err := db.AddWorkspaceMember("ws1", "bob", "member")
	if err != nil {
		t.Fatalf("first add should succeed: %v", err)
	}

	err = db.AddWorkspaceMember("ws1", "bob", "member")
	if err != ErrAlreadyMember {
		t.Fatalf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestDB_RemoveWorkspaceMember(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.AddWorkspaceMember("ws1", "bob", "member")

	err := db.RemoveWorkspaceMember("ws1", "bob")
	if err != nil {
		t.Fatalf("failed to remove member: %v", err)
	}
	members, _ := db.ListWorkspaceMembers("ws1")
	if len(members) != 1 {
		t.Fatalf("expected 1 member after remove, got %d", len(members))
	}
}

func TestDB_CreateBoard_WithStatuses(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")

	err := db.CreateBoard("bd1", "ws1", "Sprint 1", "alice", []string{"backlog", "active", "done"})
	if err != nil {
		t.Fatalf("failed to create board: %v", err)
	}

	boards, err := db.ListBoardsForUser("ws1", "alice")
	if err != nil {
		t.Fatalf("failed to list boards: %v", err)
	}
	if len(boards) != 1 {
		t.Fatalf("expected 1 board, got %d", len(boards))
	}
	if boards[0].Name != "Sprint 1" {
		t.Fatalf("expected name %q, got %q", "Sprint 1", boards[0].Name)
	}
	statuses := boards[0].StatusList()
	if len(statuses) != 3 || statuses[0] != "backlog" {
		t.Fatalf("expected custom statuses, got %v", statuses)
	}
}

func TestDB_ListBoardsForUser_AllMembersSeeAllBoards(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.AddWorkspaceMember("ws1", "bob", "member")
	db.CreateBoard("bd1", "ws1", "Board 1", "alice", model.DefaultStatuses)
	db.CreateBoard("bd2", "ws1", "Board 2", "alice", model.DefaultStatuses)

	// bob (member) sees all boards — no board-level ACL
	boards, _ := db.ListBoardsForUser("ws1", "bob")
	if len(boards) != 2 {
		t.Fatalf("expected 2 boards for bob (member sees all), got %d", len(boards))
	}

	// alice (owner) also sees all
	boards, _ = db.ListBoardsForUser("ws1", "alice")
	if len(boards) != 2 {
		t.Fatalf("expected 2 boards for alice (owner), got %d", len(boards))
	}
}

func TestDB_DeleteBoard(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.CreateBoard("bd1", "ws1", "Sprint", "alice", model.DefaultStatuses)

	err := db.DeleteBoard("bd1")
	if err != nil {
		t.Fatalf("failed to delete board: %v", err)
	}
	boards, _ := db.ListBoardsForUser("ws1", "alice")
	if len(boards) != 0 {
		t.Fatalf("expected 0 boards after delete, got %d", len(boards))
	}
}

func TestDB_UpdateBoard(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.CreateBoard("bd1", "ws1", "Sprint", "alice", model.DefaultStatuses)

	err := db.UpdateBoard("bd1", map[string]any{
		"name":     "Sprint 2",
		"statuses": "backlog,dev,review,done",
	})
	if err != nil {
		t.Fatalf("failed to update board: %v", err)
	}

	board, _ := db.GetBoard("bd1")
	if board.Name != "Sprint 2" {
		t.Fatalf("expected name Sprint 2, got %q", board.Name)
	}
	statuses := board.StatusList()
	if len(statuses) != 4 || statuses[0] != "backlog" {
		t.Fatalf("expected updated statuses, got %v", statuses)
	}
}

func TestDB_TicketsScoped_ToBoard(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.CreateBoard("bd1", "ws1", "Board 1", "alice", model.DefaultStatuses)
	db.CreateBoard("bd2", "ws1", "Board 2", "alice", model.DefaultStatuses)

	t1 := model.NewTicket("Task A", "", "alice")
	t1.BoardID = "bd1"
	db.CreateTicket(t1)

	t2 := model.NewTicket("Task B", "", "alice")
	t2.BoardID = "bd2"
	db.CreateTicket(t2)

	tickets, err := db.ListTickets("bd1", "")
	if err != nil {
		t.Fatalf("failed to list tickets: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket on bd1, got %d", len(tickets))
	}
	if tickets[0].Title != "Task A" {
		t.Fatalf("expected Task A, got %q", tickets[0].Title)
	}

	tickets, _ = db.ListTickets("bd2", "")
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket on bd2, got %d", len(tickets))
	}

	tickets, _ = db.ListTickets("", "")
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets total, got %d", len(tickets))
	}
}

func TestDB_DeleteWorkspace(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	err := db.DeleteWorkspace("ws1")
	if err != nil {
		t.Fatalf("failed to delete workspace: %v", err)
	}
	workspaces, _ := db.ListWorkspacesForUser("alice")
	if len(workspaces) != 0 {
		t.Fatalf("expected 0 workspaces after delete, got %d", len(workspaces))
	}
}

func TestDB_SearchTickets(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Fix login bug", "auth is broken", "alice")
	db.CreateTicket(t1)
	t2 := model.NewTicket("Add dashboard", "new feature", "alice")
	db.CreateTicket(t2)
	t3 := model.NewTicket("Update readme", "", "alice")
	db.CreateTicket(t3)

	tickets, err := db.SearchTickets("", "login")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 result for 'login', got %d", len(tickets))
	}
	if tickets[0].Title != "Fix login bug" {
		t.Fatalf("expected 'Fix login bug', got %q", tickets[0].Title)
	}

	tickets, _ = db.SearchTickets("", "feature")
	if len(tickets) != 1 {
		t.Fatalf("expected 1 result for 'feature', got %d", len(tickets))
	}

	tickets, _ = db.SearchTickets("", "LOGIN")
	if len(tickets) != 1 {
		t.Fatalf("expected case-insensitive match, got %d", len(tickets))
	}

	tickets, _ = db.SearchTickets("", "nonexistent")
	if len(tickets) != 0 {
		t.Fatalf("expected 0 results, got %d", len(tickets))
	}
}

func TestDB_SearchTickets_SQLWildcards(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("100% done", "", "alice")
	db.CreateTicket(t1)
	t2 := model.NewTicket("Regular task", "", "alice")
	db.CreateTicket(t2)

	tickets, err := db.SearchTickets("", "%")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 result for literal '%%', got %d", len(tickets))
	}
	if tickets[0].Title != "100% done" {
		t.Fatalf("expected '100%% done', got %q", tickets[0].Title)
	}
}

func TestDB_SearchTickets_BoardScoped(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Login bug", "", "alice")
	t1.BoardID = "bd1"
	db.CreateTicket(t1)

	t2 := model.NewTicket("Login issue", "", "alice")
	t2.BoardID = "bd2"
	db.CreateTicket(t2)

	tickets, err := db.SearchTickets("bd1", "Login")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 result scoped to bd1, got %d", len(tickets))
	}
}

func TestDB_TicketStats_DynamicKeys(t *testing.T) {
	db := newTestDB(t)
	t1 := model.NewTicket("Task 1", "", "alice")
	t1.BoardID = "b1"
	db.CreateTicket(t1)
	t2 := model.NewTicket("Task 2", "", "alice")
	t2.BoardID = "b1"
	t2.Status = model.InProgress
	db.CreateTicket(t2)

	counts, err := db.TicketStats("b1")
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}
	if counts["todo"] != 1 {
		t.Fatalf("expected 1 todo, got %d", counts["todo"])
	}
	if counts["in_progress"] != 1 {
		t.Fatalf("expected 1 in_progress, got %d", counts["in_progress"])
	}
	// done not present since no tickets have that status
	if counts["done"] != 0 {
		t.Fatalf("expected 0 done, got %d", counts["done"])
	}
}

func TestDB_PragmaSettings(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var busyTimeout int
	db.conn.Raw("PRAGMA busy_timeout").Scan(&busyTimeout)
	if busyTimeout != 5000 {
		t.Errorf("expected busy_timeout=5000, got %d", busyTimeout)
	}

	var syncMode int
	db.conn.Raw("PRAGMA synchronous").Scan(&syncMode)
	if syncMode != 1 {
		t.Errorf("expected synchronous=1 (NORMAL), got %d", syncMode)
	}

	sqlDB, err := db.conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB.Stats().MaxOpenConnections != 4 {
		t.Errorf("expected MaxOpenConnections=4, got %d", sqlDB.Stats().MaxOpenConnections)
	}
}

func TestDB_AssigneeField(t *testing.T) {
	db := newTestDB(t)
	ticket := model.NewTicket("Assigned task", "", "alice")
	ticket.Assignee = "bob"
	db.CreateTicket(ticket)

	got, err := db.GetTicket(ticket.ID)
	if err != nil {
		t.Fatalf("failed to get ticket: %v", err)
	}
	if got.Assignee != "bob" {
		t.Fatalf("expected assignee %q, got %q", "bob", got.Assignee)
	}
	if got.CreatedBy != "alice" {
		t.Fatalf("expected created_by %q, got %q", "alice", got.CreatedBy)
	}
}

func TestDB_ListTicketsMine(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "WS", "alice")
	db.CreateBoard("b1", "ws1", "Board", "alice", model.DefaultStatuses)
	db.CreateTicket(model.Ticket{ID: "t1", BoardID: "b1", Title: "Alice's", CreatedBy: "alice"})
	db.CreateTicket(model.Ticket{ID: "t2", BoardID: "b1", Title: "Bob's", CreatedBy: "bob"})
	db.CreateTicket(model.Ticket{ID: "t3", BoardID: "b1", Title: "Assigned", CreatedBy: "bob", Assignee: "alice"})

	tickets, err := db.ListTicketsMine("b1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 2 {
		t.Errorf("expected 2 tickets for alice, got %d", len(tickets))
	}
}

func TestDB_TicketsSurviveRestart(t *testing.T) {
	// Simulates what happens on Railway deploy: server stops, new container
	// starts, NewDB() runs AutoMigrate again on the same database file.
	dbPath := t.TempDir() + "/raptor.db"

	// First "boot": create workspace, board, and tickets
	db1, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.CreateWorkspace("ws1", "Team", "alice")
	db1.CreateBoard("bd1", "ws1", "Sprint", "alice", model.DefaultStatuses)

	ticket := model.NewTicket("Important task", "do not lose me", "alice")
	ticket.BoardID = "bd1"
	db1.CreateTicket(ticket)

	tickets, _ := db1.ListTickets("bd1", "")
	if len(tickets) != 1 {
		t.Fatalf("setup: expected 1 ticket, got %d", len(tickets))
	}
	db1.Close()

	// Second "boot": simulates redeploy — NewDB runs AutoMigrate again
	db2, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()

	tickets, err = db2.ListTickets("bd1", "")
	if err != nil {
		t.Fatalf("failed to list after restart: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket after restart, got %d (tickets lost!)", len(tickets))
	}
	if tickets[0].Title != "Important task" {
		t.Fatalf("expected title %q, got %q", "Important task", tickets[0].Title)
	}
}

func TestDB_TicketsSurviveRestart_WithForeignKeysOn(t *testing.T) {
	// Regression: even if foreign_keys was left ON from a previous connection
	// (e.g. unclean shutdown), AutoMigrate must not lose data.
	dbPath := t.TempDir() + "/raptor.db"

	// First boot: create data
	db1, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.CreateWorkspace("ws1", "Team", "alice")
	db1.CreateBoard("bd1", "ws1", "Sprint", "alice", model.DefaultStatuses)
	ticket := model.NewTicket("Survive me", "please", "alice")
	ticket.BoardID = "bd1"
	db1.CreateTicket(ticket)

	// Leave foreign_keys ON (simulating unclean state persisted in DB)
	db1.conn.Exec("PRAGMA foreign_keys = ON")
	db1.Close()

	// Second boot: NewDB must handle this safely
	db2, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()

	tickets, err := db2.ListTickets("bd1", "")
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket after restart with FK on, got %d (tickets lost!)", len(tickets))
	}
}

func TestDB_ForeignKeysOffDuringMigration(t *testing.T) {
	// Foreign keys MUST be OFF during AutoMigrate to prevent CASCADE deletes
	// when GORM recreates tables. Verify NewDB explicitly disables them first.
	dbPath := t.TempDir() + "/raptor.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// After NewDB, foreign_keys should be ON (enabled after migration)
	var fk int
	db.conn.Raw("PRAGMA foreign_keys").Scan(&fk)
	if fk != 1 {
		t.Fatalf("expected foreign_keys=1 after NewDB, got %d", fk)
	}
}

func TestDB_ExplicitForeignKeysOff_BeforeMigration(t *testing.T) {
	// NewDB must explicitly set PRAGMA foreign_keys = OFF before AutoMigrate,
	// not rely on the SQLite default. This prevents data loss if a future
	// driver version changes the default or if constraints are added later.
	dbPath := t.TempDir() + "/raptor.db"
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Verify foreign_keys is ON after NewDB completes (set after migration)
	var fk int
	db.conn.Raw("PRAGMA foreign_keys").Scan(&fk)
	if fk != 1 {
		t.Fatalf("expected foreign_keys=1 after NewDB, got %d", fk)
	}
}

func TestDB_CascadeDeleteHandledInAppCode(t *testing.T) {
	// Verify that deleting a board via app code still cascades to tickets,
	// even without constraint:OnDelete:CASCADE in GORM tags.
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "Team", "alice")
	db.CreateBoard("bd1", "ws1", "Sprint", "alice", model.DefaultStatuses)

	t1 := model.NewTicket("Task 1", "", "alice")
	t1.BoardID = "bd1"
	db.CreateTicket(t1)
	t2 := model.NewTicket("Task 2", "", "alice")
	t2.BoardID = "bd1"
	db.CreateTicket(t2)

	err := db.DeleteBoard("bd1")
	if err != nil {
		t.Fatalf("failed to delete board: %v", err)
	}

	tickets, _ := db.ListTickets("bd1", "")
	if len(tickets) != 0 {
		t.Fatalf("expected 0 tickets after board delete, got %d (app cascade broken!)", len(tickets))
	}
}

func TestDB_DeleteWorkspace_CascadesAll(t *testing.T) {
	db := newTestDB(t)
	db.CreateWorkspace("ws1", "WS", "alice")
	db.CreateBoard("b1", "ws1", "Board1", "alice", model.DefaultStatuses)
	db.CreateBoard("b2", "ws1", "Board2", "alice", model.DefaultStatuses)
	db.CreateTicket(model.Ticket{ID: "t1", BoardID: "b1", Title: "T1"})
	db.CreateTicket(model.Ticket{ID: "t2", BoardID: "b2", Title: "T2"})

	err := db.DeleteWorkspace("ws1")
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	db.conn.Model(&model.Ticket{}).Where("board_id IN ?", []string{"b1", "b2"}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 tickets after cascade delete, got %d", count)
	}
}
