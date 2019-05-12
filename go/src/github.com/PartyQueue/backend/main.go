package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
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
	Id        string `sql:",primary"`
	HostToken string `graphql:"-"`
	Created   time.Time
	LastUsed  time.Time
}

type Request struct {
	Uri      string `sql:",primary"`
	Priority int
	Time     time.Time
	RoomId   string   `sql:",primary"`
	Metadata Metadata `sql:"-"`
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
	// posts returns all posts in the database.
	query.FieldFunc("rooms", func(ctx context.Context) ([]*Room, error) {
		var rooms []*Room
		if err := s.db.Query(ctx, &rooms, nil, nil); err != nil {
			return nil, err
		}
		return rooms, nil
	})

	query.FieldFunc("requests", func(ctx context.Context) ([]*Request, error) {
		var requests []*Request
		if err := s.db.Query(ctx, &requests, nil, nil); err != nil {
			return nil, err
		}
		return requests, nil
	})
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
		return room, nil
	})
}

func buildSqlgenSchema() *sqlgen.Schema {
	schema := sqlgen.NewSchema()
	schema.MustRegisterType("rooms", sqlgen.UniqueId, Room{})
	schema.MustRegisterType("requests", sqlgen.UniqueId, Request{})
	schema.MustRegisterType("songs", sqlgen.UniqueId, Metadata{})
	return schema
}

var SqlgenSchema = buildSqlgenSchema()

// schema builds the graphql schema.
func (s *Server) schema() *graphql.Schema {
	builder := schemabuilder.NewSchema()
	s.registerQuery(builder)
	s.registerMutation(builder)
	s.registerRoom(builder)
	return builder.MustBuild()
}

type DatabaseConfig struct {
	Username     string
	Password     string
	Hostname     string
	DatabaseName string
	Port         string
}

// SQLMultiValue creates a multi value interpolation statement:
// in: 3 out: (?,?,?)
// in: 6 out: (?,?,?,?,?,?)
func SQLMultiValue(valueCount int) string {
	var builder strings.Builder
	_ = builder.WriteByte('(')
	for i := 0; i < valueCount; i++ {
		if i != 0 {
			_ = builder.WriteByte(',')
		}
		_ = builder.WriteByte('?')
	}
	builder.WriteByte(')')
	return builder.String()
}

// registerMutation registers the root mutation type.
func (s *Server) registerMutation(schema *schemabuilder.Schema) {
	object := schema.Mutation()

	object.FieldFunc("echo", func(ctx context.Context, args struct{ Text string }) (string, error) {
		return args.Text, nil
	})

}

func (s *Server) registerRoom(schema *schemabuilder.Schema) {
	obj := schema.Object("Room", Room{})
	obj.FieldFunc("requests", func(ctx context.Context, p *Room) ([]*Request, error) {
		var requests []*Request
		if err := s.db.Query(ctx, &requests, sqlgen.Filter{"room_id": p.Id}, &sqlgen.SelectOptions{OrderBy: "priority,time"}); err != nil {
			return nil, nil
		}
		var songs []*Metadata

		valuesLen := len(requests)
		if valuesLen == 0 {
			return nil, nil
		}
		// hacky join
		whereStr := fmt.Sprintf("uri IN %s", SQLMultiValue(valuesLen))
		valuesSlice := make([]interface{}, valuesLen)
		for i := range requests {
			valuesSlice[i] = requests[i].Uri
		}
		if err := s.db.Query(ctx, &songs, nil, &sqlgen.SelectOptions{Where: whereStr, Values: valuesSlice}); err != nil {
			return nil, nil
		}
		m := make(map[string]*Metadata)
		for _, song := range songs {
			m[song.Uri] = song
		}
		for _, element := range requests {
			element.Metadata = *m[element.Uri]
		}
		return requests, nil
	})
}

func main() {

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
