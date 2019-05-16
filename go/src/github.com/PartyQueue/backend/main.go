package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/samsarahq/thunder/graphql"
	"github.com/samsarahq/thunder/graphql/graphiql"
	"github.com/samsarahq/thunder/graphql/introspection"
	"github.com/samsarahq/thunder/graphql/schemabuilder"
	"github.com/samsarahq/thunder/livesql"
	"github.com/samsarahq/thunder/sqlgen"
)

// A Post holds a row from the MySQL posts table.
type Room struct {
	Id              string `sql:",primary"`
	HostToken       string `graphql:"-"`
	Created         time.Time
	LastUsed        time.Time
	StartedAt       time.Time     `graphql:"-"`
	PausedAt        time.Time     `graphql:"-"`
	CurrentDuration time.Duration `graphql:"-"`
	IsPaused        bool          `sql:"-"`
}

type Request struct {
	Uri      string `sql:",primary"`
	Priority int    `graphql:"-"`
	Time     time.Time
	RoomId   string `sql:",primary"`
}

type Metadata struct {
	Uri        string `sql:",primary"`
	Title      string
	Artist     string
	Popularity int       `graphql:"-"`
	LastReq    time.Time `graphql:"-"`
	Image      string
}

type roomIdArgs struct {
	Id string
}

// Server implements a graphql server. It has persistent handles to eg. the
// database.
type Server struct {
	db *livesql.LiveDB
}

// registerQuery registers the root query resolvers.
func (s *Server) registerQuery(schema *schemabuilder.Schema) {
	query := schema.Query()

	query.FieldFunc("room", func(ctx context.Context, args roomIdArgs) (*Room, error) {
		var room *Room
		filter := sqlgen.Filter{}
		filter["id"] = args.Id
		if err := s.db.QueryRow(ctx, &room, filter, nil); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		room.IsPaused = room.PausedAt != time.Time{}
		return room, nil
	})
}

// registerMutation registers the root mutation type.
func (s *Server) registerMutation(schema *schemabuilder.Schema) {
	object := schema.Mutation()

	// temporary
	object.FieldFunc("echo", func(ctx context.Context, args struct{ Text string }) (string, error) {
		return args.Text, nil
	})

}

func (s *Server) registerRoom(schema *schemabuilder.Schema) {
	obj := schema.Object("Room", Room{})

	obj.FieldFunc("nowPlaying", func(ctx context.Context, p *Room) (*Request, error) {
		var nowPlaying *Request
		if err := s.db.QueryRow(ctx, &nowPlaying, sqlgen.Filter{"room_id": p.Id}, &sqlgen.SelectOptions{OrderBy: "priority,time", Limit: 1, Where: "priority < 0"}); err != nil {
			return nil, nil
		}
		return nowPlaying, nil
	})

	obj.FieldFunc("requests", func(ctx context.Context, p *Room) ([]*Request, error) {
		var requests []*Request
		if err := s.db.Query(ctx, &requests, sqlgen.Filter{"room_id": p.Id}, &sqlgen.SelectOptions{OrderBy: "priority,time", Where: "priority >= 0"}); err != nil {
			return nil, nil
		}
		return requests, nil
	})

	obj.FieldFunc("remainingMs", func(ctx context.Context, r *Room) (*float64, error) {
		var remaining float64
		if r.IsPaused {
			remaining = r.CurrentDuration.Seconds()
		} else {
			remaining = time.Until(r.StartedAt.Add(r.CurrentDuration)).Seconds()
		}
		remaining = math.Max(remaining, 0) * 1000
		return &remaining, nil
	})
}

func (s *Server) registerRequest(schema *schemabuilder.Schema) {
	obj := schema.Object("Request", Request{})
	obj.FieldFunc("metadata", func(ctx context.Context, r *Request) (*Metadata, error) {
		var metadata *Metadata
		if err := s.db.QueryRow(ctx, &metadata, sqlgen.Filter{"uri": r.Uri}, nil); err != nil {
			return nil, nil
		}
		return metadata, nil
	})
}

func buildSqlgenSchema() *sqlgen.Schema {
	schema := sqlgen.NewSchema()
	schema.MustRegisterType("rooms", sqlgen.UniqueId, Room{})
	schema.MustRegisterType("requests", sqlgen.UniqueId, Request{})
	schema.MustRegisterType("songs", sqlgen.UniqueId, Metadata{})
	return schema
}

// schema builds the graphql schema.
func (s *Server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	s.registerQuery(builder)
	s.registerMutation(builder)
	s.registerRoom(builder)
	s.registerRequest(builder)
	return builder.MustBuild()
}

func main() {
	SqlgenSchema := buildSqlgenSchema()

	db, err := livesql.Open("localhost", 3307, "root", "dev", "party_queue", SqlgenSchema)
	if err != nil {
		fmt.Println(err.Error())
	}
	server := &Server{
		db: db,
	}

	schema := server.schema()
	introspection.AddIntrospectionToSchema(schema)

	// Expose schema and graphiql.
	http.Handle("/graphql", graphql.Handler(schema))
	http.Handle("/graphiql/", http.StripPrefix("/graphiql/", graphiql.Handler()))
	http.ListenAndServe(":3030", nil)
}
