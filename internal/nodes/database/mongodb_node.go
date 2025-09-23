package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/n8n-go/internal/core/base"
	"github.com/n8n-go/internal/core/interfaces"
)

// MongoDBNode provides MongoDB database operations
type MongoDBNode struct {
	*base.BaseNode
	client     *mongo.Client
	connection string
}

// NewMongoDBNode creates a new MongoDB node
func NewMongoDBNode() *MongoDBNode {
	return &MongoDBNode{
		BaseNode: base.NewBaseNode("MongoDB", "MongoDB Operations"),
	}
}

// GetMetadata returns the node metadata
func (n *MongoDBNode) GetMetadata() interfaces.NodeMetadata {
	return interfaces.NodeMetadata{
		Name:        "MongoDB",
		DisplayName: "MongoDB",
		Description: "Read, write, and query data in MongoDB",
		Group:       []string{"Database"},
		Version:     1,
		Inputs:      []string{"main"},
		Outputs:     []string{"main"},
		Credentials: []interfaces.CredentialType{
			{
				Name:        "mongoDb",
				Required:    true,
				DisplayName: "MongoDB",
			},
		},
		Properties: []interfaces.NodeProperty{
			{
				Name:        "operation",
				DisplayName: "Operation",
				Type:        "options",
				Options: []interfaces.OptionItem{
					{Name: "Find", Value: "find"},
					{Name: "Find One", Value: "findOne"},
					{Name: "Insert", Value: "insert"},
					{Name: "Insert Many", Value: "insertMany"},
					{Name: "Update", Value: "update"},
					{Name: "Update Many", Value: "updateMany"},
					{Name: "Replace", Value: "replace"},
					{Name: "Delete", Value: "delete"},
					{Name: "Delete Many", Value: "deleteMany"},
					{Name: "Aggregate", Value: "aggregate"},
					{Name: "Count", Value: "count"},
					{Name: "Distinct", Value: "distinct"},
				},
				Default:     "find",
				Required:    true,
				Description: "The operation to perform",
			},
			{
				Name:        "database",
				DisplayName: "Database",
				Type:        "string",
				Default:     "",
				Required:    true,
				Description: "The database name",
			},
			{
				Name:        "collection",
				DisplayName: "Collection",
				Type:        "string",
				Default:     "",
				Required:    true,
				Description: "The collection name",
			},
			{
				Name:        "query",
				DisplayName: "Query",
				Type:        "json",
				Default:     "{}",
				Description: "MongoDB query in JSON format",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"find", "findOne", "update", "updateMany", "delete", "deleteMany", "count"},
					},
				},
			},
			{
				Name:        "fields",
				DisplayName: "Fields/Projection",
				Type:        "json",
				Default:     "{}",
				Description: "Fields to return (projection)",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"find", "findOne"},
					},
				},
			},
			{
				Name:        "sort",
				DisplayName: "Sort",
				Type:        "json",
				Default:     "{}",
				Description: "Sort specification",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"find"},
					},
				},
			},
			{
				Name:        "limit",
				DisplayName: "Limit",
				Type:        "number",
				Default:     10,
				Description: "Maximum number of documents to return",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"find"},
					},
				},
			},
			{
				Name:        "skip",
				DisplayName: "Skip",
				Type:        "number",
				Default:     0,
				Description: "Number of documents to skip",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"find"},
					},
				},
			},
			{
				Name:        "document",
				DisplayName: "Document",
				Type:        "json",
				Default:     "{}",
				Description: "Document to insert or update",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"insert", "replace"},
					},
				},
			},
			{
				Name:        "documents",
				DisplayName: "Documents",
				Type:        "json",
				Default:     "[]",
				Description: "Array of documents to insert",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"insertMany"},
					},
				},
			},
			{
				Name:        "updateData",
				DisplayName: "Update Data",
				Type:        "json",
				Default:     "{}",
				Description: "Update operations ($set, $inc, etc.)",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"update", "updateMany"},
					},
				},
			},
			{
				Name:        "pipeline",
				DisplayName: "Aggregation Pipeline",
				Type:        "json",
				Default:     "[]",
				Description: "MongoDB aggregation pipeline",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"aggregate"},
					},
				},
			},
			{
				Name:        "upsert",
				DisplayName: "Upsert",
				Type:        "boolean",
				Default:     false,
				Description: "Create document if it doesn't exist",
				DisplayOptions: map[string]interface{}{
					"show": map[string]interface{}{
						"operation": []string{"update", "updateMany"},
					},
				},
			},
		},
	}
}

// Execute runs the MongoDB operation
func (n *MongoDBNode) Execute(ctx context.Context, params interfaces.ExecutionParams) (interfaces.NodeOutput, error) {
	// Get credentials
	credentials, err := params.GetCredentials("mongoDb")
	if err != nil {
		return interfaces.NodeOutput{}, fmt.Errorf("failed to get MongoDB credentials: %w", err)
	}

	// Connect to MongoDB
	if err := n.connect(credentials); err != nil {
		return interfaces.NodeOutput{}, err
	}
	defer n.disconnect()

	// Get parameters
	operation := params.GetNodeParameter("operation", "find").(string)
	database := params.GetNodeParameter("database", "").(string)
	collection := params.GetNodeParameter("collection", "").(string)

	if database == "" || collection == "" {
		return interfaces.NodeOutput{}, fmt.Errorf("database and collection are required")
	}

	// Get collection handle
	coll := n.client.Database(database).Collection(collection)

	// Execute operation
	var result interface{}
	switch operation {
	case "find":
		result, err = n.executeFind(coll, params)
	case "findOne":
		result, err = n.executeFindOne(coll, params)
	case "insert":
		result, err = n.executeInsert(coll, params)
	case "insertMany":
		result, err = n.executeInsertMany(coll, params)
	case "update":
		result, err = n.executeUpdate(coll, params)
	case "updateMany":
		result, err = n.executeUpdateMany(coll, params)
	case "replace":
		result, err = n.executeReplace(coll, params)
	case "delete":
		result, err = n.executeDelete(coll, params)
	case "deleteMany":
		result, err = n.executeDeleteMany(coll, params)
	case "aggregate":
		result, err = n.executeAggregate(coll, params)
	case "count":
		result, err = n.executeCount(coll, params)
	case "distinct":
		result, err = n.executeDistinct(coll, params)
	default:
		err = fmt.Errorf("unsupported operation: %s", operation)
	}

	if err != nil {
		return interfaces.NodeOutput{}, err
	}

	// Format output
	var outputItems []interfaces.ItemData
	switch v := result.(type) {
	case []map[string]interface{}:
		for i, doc := range v {
			outputItems = append(outputItems, interfaces.ItemData{
				JSON:  doc,
				Index: i,
			})
		}
	case map[string]interface{}:
		outputItems = append(outputItems, interfaces.ItemData{
			JSON:  v,
			Index: 0,
		})
	default:
		outputItems = append(outputItems, interfaces.ItemData{
			JSON: map[string]interface{}{
				"result": result,
			},
			Index: 0,
		})
	}

	return interfaces.NodeOutput{
		Items: outputItems,
	}, nil
}

// Connection methods

func (n *MongoDBNode) connect(credentials map[string]interface{}) error {
	// Build connection string
	connStr := ""
	if uri, ok := credentials["connectionString"].(string); ok && uri != "" {
		connStr = uri
	} else {
		// Build from components
		host := credentials["host"].(string)
		port := credentials["port"].(int)
		user := credentials["user"].(string)
		password := credentials["password"].(string)
		authDb := "admin"
		if db, ok := credentials["authDatabase"].(string); ok && db != "" {
			authDb = db
		}

		if user != "" && password != "" {
			connStr = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", user, password, host, port, authDb)
		} else {
			connStr = fmt.Sprintf("mongodb://%s:%d", host, port)
		}
	}

	// Create client
	clientOptions := options.Client().ApplyURI(connStr)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	n.client = client
	n.connection = connStr
	return nil
}

func (n *MongoDBNode) disconnect() {
	if n.client != nil {
		n.client.Disconnect(context.TODO())
	}
}

// Operation implementations

func (n *MongoDBNode) executeFind(coll *mongo.Collection, params interfaces.ExecutionParams) ([]map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))
	projection := n.parseJSON(params.GetNodeParameter("fields", "{}").(string))
	sortSpec := n.parseJSON(params.GetNodeParameter("sort", "{}").(string))
	limit := int64(params.GetNodeParameter("limit", 10).(float64))
	skip := int64(params.GetNodeParameter("skip", 0).(float64))

	opts := options.Find()
	if len(projection) > 0 {
		opts.SetProjection(projection)
	}
	if len(sortSpec) > 0 {
		opts.SetSort(sortSpec)
	}
	opts.SetLimit(limit)
	opts.SetSkip(skip)

	cursor, err := coll.Find(context.TODO(), query, opts)
	if err != nil {
		return nil, fmt.Errorf("find operation failed: %w", err)
	}
	defer cursor.Close(context.TODO())

	var results []map[string]interface{}
	for cursor.Next(context.TODO()) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		results = append(results, doc)
	}

	return results, nil
}

func (n *MongoDBNode) executeFindOne(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))
	projection := n.parseJSON(params.GetNodeParameter("fields", "{}").(string))

	opts := options.FindOne()
	if len(projection) > 0 {
		opts.SetProjection(projection)
	}

	var result map[string]interface{}
	err := coll.FindOne(context.TODO(), query, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("findOne operation failed: %w", err)
	}

	return result, nil
}

func (n *MongoDBNode) executeInsert(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	doc := n.parseJSON(params.GetNodeParameter("document", "{}").(string))
	
	result, err := coll.InsertOne(context.TODO(), doc)
	if err != nil {
		return nil, fmt.Errorf("insert operation failed: %w", err)
	}

	return map[string]interface{}{
		"insertedId": result.InsertedID,
	}, nil
}

func (n *MongoDBNode) executeInsertMany(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	docsJSON := params.GetNodeParameter("documents", "[]").(string)
	var docs []interface{}
	if err := json.Unmarshal([]byte(docsJSON), &docs); err != nil {
		return nil, fmt.Errorf("invalid documents JSON: %w", err)
	}

	result, err := coll.InsertMany(context.TODO(), docs)
	if err != nil {
		return nil, fmt.Errorf("insertMany operation failed: %w", err)
	}

	return map[string]interface{}{
		"insertedIds":   result.InsertedIDs,
		"insertedCount": len(result.InsertedIDs),
	}, nil
}

func (n *MongoDBNode) executeUpdate(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))
	updateData := n.parseJSON(params.GetNodeParameter("updateData", "{}").(string))
	upsert := params.GetNodeParameter("upsert", false).(bool)

	opts := options.Update().SetUpsert(upsert)
	result, err := coll.UpdateOne(context.TODO(), query, updateData, opts)
	if err != nil {
		return nil, fmt.Errorf("update operation failed: %w", err)
	}

	return map[string]interface{}{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
		"upsertedCount": result.UpsertedCount,
		"upsertedId":    result.UpsertedID,
	}, nil
}

func (n *MongoDBNode) executeUpdateMany(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))
	updateData := n.parseJSON(params.GetNodeParameter("updateData", "{}").(string))
	upsert := params.GetNodeParameter("upsert", false).(bool)

	opts := options.Update().SetUpsert(upsert)
	result, err := coll.UpdateMany(context.TODO(), query, updateData, opts)
	if err != nil {
		return nil, fmt.Errorf("updateMany operation failed: %w", err)
	}

	return map[string]interface{}{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
		"upsertedCount": result.UpsertedCount,
	}, nil
}

func (n *MongoDBNode) executeReplace(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))
	document := n.parseJSON(params.GetNodeParameter("document", "{}").(string))

	result, err := coll.ReplaceOne(context.TODO(), query, document)
	if err != nil {
		return nil, fmt.Errorf("replace operation failed: %w", err)
	}

	return map[string]interface{}{
		"matchedCount":  result.MatchedCount,
		"modifiedCount": result.ModifiedCount,
		"upsertedCount": result.UpsertedCount,
		"upsertedId":    result.UpsertedID,
	}, nil
}

func (n *MongoDBNode) executeDelete(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))

	result, err := coll.DeleteOne(context.TODO(), query)
	if err != nil {
		return nil, fmt.Errorf("delete operation failed: %w", err)
	}

	return map[string]interface{}{
		"deletedCount": result.DeletedCount,
	}, nil
}

func (n *MongoDBNode) executeDeleteMany(coll *mongo.Collection, params interfaces.ExecutionParams) (map[string]interface{}, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))

	result, err := coll.DeleteMany(context.TODO(), query)
	if err != nil {
		return nil, fmt.Errorf("deleteMany operation failed: %w", err)
	}

	return map[string]interface{}{
		"deletedCount": result.DeletedCount,
	}, nil
}

func (n *MongoDBNode) executeAggregate(coll *mongo.Collection, params interfaces.ExecutionParams) ([]map[string]interface{}, error) {
	pipelineJSON := params.GetNodeParameter("pipeline", "[]").(string)
	var pipeline []bson.M
	if err := json.Unmarshal([]byte(pipelineJSON), &pipeline); err != nil {
		return nil, fmt.Errorf("invalid pipeline JSON: %w", err)
	}

	cursor, err := coll.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate operation failed: %w", err)
	}
	defer cursor.Close(context.TODO())

	var results []map[string]interface{}
	for cursor.Next(context.TODO()) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		results = append(results, doc)
	}

	return results, nil
}

func (n *MongoDBNode) executeCount(coll *mongo.Collection, params interfaces.ExecutionParams) (int64, error) {
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))

	count, err := coll.CountDocuments(context.TODO(), query)
	if err != nil {
		return 0, fmt.Errorf("count operation failed: %w", err)
	}

	return count, nil
}

func (n *MongoDBNode) executeDistinct(coll *mongo.Collection, params interfaces.ExecutionParams) ([]interface{}, error) {
	field := params.GetNodeParameter("field", "").(string)
	query := n.parseJSON(params.GetNodeParameter("query", "{}").(string))

	if field == "" {
		return nil, fmt.Errorf("field is required for distinct operation")
	}

	results, err := coll.Distinct(context.TODO(), field, query)
	if err != nil {
		return nil, fmt.Errorf("distinct operation failed: %w", err)
	}

	return results, nil
}

// Helper methods

func (n *MongoDBNode) parseJSON(jsonStr string) bson.M {
	var result bson.M
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return bson.M{}
	}
	return result
}

// Clone creates a copy of the node
func (n *MongoDBNode) Clone() interfaces.Node {
	return &MongoDBNode{
		BaseNode: n.BaseNode.Clone(),
	}
}