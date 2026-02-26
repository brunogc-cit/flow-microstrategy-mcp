package server_test

import (
	"fmt"
	"testing"

	analytics "github.com/brunogc-cit/flow-microstrategy-mcp/internal/analytics/mocks"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/config"
	db "github.com/brunogc-cit/flow-microstrategy-mcp/internal/database/mocks"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

func TestToolRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aService := analytics.NewMockService(ctrl)
	aService.EXPECT().IsEnabled().AnyTimes().Return(true)
	aService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	aService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	aService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	t.Run("verifies expected tools are registered", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// 2 Cypher (get-schema, read-cypher) + 1 GDS (list-gds-procedures) + 4 MSTR = 7 total
		expectedTotalToolsCount := 7

		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should register only readonly tools when readonly", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      true,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// All tools are readonly, so all 7 tools are registered
		expectedTotalToolsCount := 7

		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should register all tools when readonly is set to false", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      false,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// 2 Cypher + 1 GDS + 4 MSTR = 7 total (no write tools exist)
		expectedTotalToolsCount := 7

		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should remove GDS tools if GDS is not present", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, false)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      false,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// 2 Cypher + 4 MSTR = 6 total (list-gds-procedures excluded)
		expectedTotalToolsCount := 6

		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
}

// utility to mock the invocation required by VerifyRequirements
func getMockedDBService(ctrl *gomock.Controller, withGDS bool) *db.MockService {
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
		{
			Keys: []string{"first"},
			Values: []any{
				int64(1),
			},
		},
	}, nil)
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
		{
			Keys: []string{"apocMetaSchemaAvailable"},
			Values: []any{
				bool(true),
			},
		},
	}, nil)
	gdsVersionQuery := "RETURN gds.version() as gdsVersion"
	if withGDS {
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"gdsVersion"},
				Values: []any{
					string("2.22.0"),
				},
			},
		}, nil)
		return mockDB
	}
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return(nil, fmt.Errorf("Unknown function 'gds.version'"))

	return mockDB
}
