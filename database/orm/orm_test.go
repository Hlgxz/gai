package orm_test

import (
	"context"
	"testing"

	"github.com/Hlgxz/gai/database"
	_ "github.com/Hlgxz/gai/database/driver"
	"github.com/Hlgxz/gai/database/orm"
)

type User struct {
	orm.Model
	Name  string
	Email string
	Age   int
}

type Comment struct {
	orm.Model
	PostId uint64 `gai:"column:post_id"`
	Body   string `gai:"column:body"`
}

type Post struct {
	orm.Model
	UserId   uint64    `gai:"column:user_id"`
	Title    string    `gai:"column:title"`
	Comments []Comment `gai:"hasMany;fk:post_id"`
}

type UserWithPosts struct {
	orm.Model
	Name  string
	Posts []Post `gai:"hasMany;fk:user_id"`
}

func (UserWithPosts) TableName() string { return "users" }

func openTestDB(t *testing.T) *orm.DB {
	t.Helper()
	db, err := database.Open(database.Config{Driver: "sqlite", DSN: ":memory:"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(context.Background(), `CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT, email TEXT, age INTEGER,
		created_at TEXT, updated_at TEXT, deleted_at TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCRUDAndTransaction(t *testing.T) {
	db := openTestDB(t)

	u, err := orm.Create[User](db, &User{Name: "Ada", Email: "ada@test.com", Age: 30})
	if err != nil || u.ID == 0 {
		t.Fatalf("create: %+v %v", u, err)
	}

	found, err := orm.Find[User](db, u.ID)
	if err != nil || found == nil || found.Name != "Ada" {
		t.Fatalf("find: %+v %v", found, err)
	}

	found.Age = 31
	if err := orm.Update[User](db, found); err != nil {
		t.Fatal(err)
	}

	err = db.Transaction(context.Background(), func(tx *orm.DB) error {
		_, err := orm.Create[User](tx, &User{Name: "Rollback", Email: "r@t.com"})
		if err != nil {
			return err
		}
		return errBoom
	})
	if err == nil {
		t.Fatal("expected rollback")
	}
	n, _ := orm.Count(orm.Query[User](db).Where("name", "=", "Rollback"))
	if n != 0 {
		t.Fatalf("rolled back row still present: %d", n)
	}
}

func TestFirstOrCreateJoinSoftDelete(t *testing.T) {
	db := openTestDB(t)
	_, _ = db.Exec(context.Background(), `CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER, title TEXT,
		created_at TEXT, updated_at TEXT, deleted_at TEXT
	)`)

	u, _ := orm.Create[User](db, &User{Name: "Bob", Email: "b@t.com", Age: 20})
	_, err := db.Exec(context.Background(), `INSERT INTO posts (user_id, title, created_at, updated_at) VALUES (?,?,datetime('now'),datetime('now'))`, u.ID, "Hello")
	if err != nil {
		t.Fatal(err)
	}

	again, err := orm.FirstOrCreate[User](orm.Query[User](db).Where("email", "=", "b@t.com"), &User{Name: "X", Email: "b@t.com"})
	if err != nil || again.Name != "Bob" {
		t.Fatalf("first or create: %+v %v", again, err)
	}

	type row struct {
		Name  string `gai:"column:name"`
		Title string `gai:"column:title"`
	}
	q := orm.Table(db, "users").Join("posts", "posts.user_id", "=", "users.id").Select("users.name", "posts.title")
	sql, _ := q.ToSQL()
	if sql == "" {
		t.Fatal("empty sql")
	}

	if err := orm.Delete[User](db, u); err != nil {
		t.Fatal(err)
	}
	gone, _ := orm.First[User](orm.Query[User](db).Where("id", "=", u.ID))
	if gone != nil {
		t.Fatal("soft delete should hide row")
	}
	if err := orm.ForceDelete[User](db, u); err != nil {
		t.Fatal(err)
	}
}

func TestAggregates(t *testing.T) {
	db := openTestDB(t)
	_ = orm.Insert[User](db, []User{
		{Name: "a", Age: 10},
		{Name: "b", Age: 20},
	})
	sum, err := orm.Sum(orm.Query[User](db), "age")
	if err != nil || sum != 30 {
		t.Fatalf("sum=%v err=%v", sum, err)
	}
	ok, _ := orm.Exists(orm.Query[User](db).Where("name", "=", "a"))
	if !ok {
		t.Fatal("exists")
	}
}

func TestUpdateAllAndNestedWith(t *testing.T) {
	db := openTestDB(t)
	_, _ = db.Exec(context.Background(), `CREATE TABLE posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER, title TEXT,
		created_at TEXT, updated_at TEXT, deleted_at TEXT
	)`)
	_, _ = db.Exec(context.Background(), `CREATE TABLE comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER, body TEXT,
		created_at TEXT, updated_at TEXT, deleted_at TEXT
	)`)

	u, err := orm.Create[User](db, &User{Name: "Eve", Email: "e@t.com", Age: 1})
	if err != nil {
		t.Fatal(err)
	}
	n, err := orm.Query[User](db).Where("id", "=", u.ID).UpdateAll(map[string]any{"age": 42})
	if err != nil || n != 1 {
		t.Fatalf("update all n=%d err=%v", n, err)
	}
	sql, _ := orm.Query[User](db).ForUpdate().ToSQL()
	if sql == "" {
		t.Fatal("for update sql")
	}

	_, err = db.Exec(context.Background(), `INSERT INTO posts (user_id, title, created_at, updated_at) VALUES (?,?,datetime('now'),datetime('now'))`, u.ID, "P1")
	if err != nil {
		t.Fatal(err)
	}
	posts, err := orm.Get[Post](orm.Query[Post](db).Where("user_id", "=", u.ID))
	if err != nil || len(posts) != 1 {
		t.Fatalf("posts %+v %v", posts, err)
	}
	_, err = db.Exec(context.Background(), `INSERT INTO comments (post_id, body, created_at, updated_at) VALUES (?,?,datetime('now'),datetime('now'))`, posts[0].ID, "hi")
	if err != nil {
		t.Fatal(err)
	}

	users, err := orm.Get[UserWithPosts](orm.Query[UserWithPosts](db).Where("id", "=", u.ID))
	if err != nil {
		t.Fatal(err)
	}
	if err := orm.With(db, users, "Posts.Comments"); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || len(users[0].Posts) != 1 || len(users[0].Posts[0].Comments) != 1 {
		t.Fatalf("nested: %+v", users)
	}
}

type errBoomT string

func (e errBoomT) Error() string { return string(e) }

var errBoom error = errBoomT("boom")
