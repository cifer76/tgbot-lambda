package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	opensearch "github.com/opensearch-project/opensearch-go"
	opensearchapi "github.com/opensearch-project/opensearch-go/opensearchapi"
)

const indexName = "groups"

var opensvc *opensearch.Client

func opensearchWriteGroup(ctx context.Context, r GroupRecord) {
	// Add a document to the index.
	s, _ := json.Marshal(r)
	document := strings.NewReader(string(s))
	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: strconv.FormatInt(r.ChatID, 10),
		Body:       document,
	}

	insertResponse, err := req.Do(context.Background(), opensvc)
	if err != nil {
		fmt.Println("failed to insert document ", err)
		return
	}
	fmt.Println(insertResponse)
}

func opensearchSearchGroup(ctx context.Context, keywords []string) []GroupRecord {
	// Search for the document.
	query := strings.Join(keywords, " ")
	content := `{
        "size": 10,
        "query": {
            "multi_match": {
                "query": "%s",
                "fields": ["title", "description"]
            }
        }
    }`
	content = fmt.Sprintf(content, query)

	search := opensearchapi.SearchRequest{
		Index: []string{indexName},
		Body:  strings.NewReader(content),
	}

	searchResponse, err := search.Do(context.Background(), opensvc)
	if err != nil {
		fmt.Println("failed to search document ", err)
		os.Exit(1)
	}
	fmt.Println(searchResponse)

	groups := []GroupRecord{}
	if searchResponse.IsError() {
		return groups
	}

	resp := map[string]interface{}{}
	_ = json.Unmarshal([]byte(searchResponse.String()), &resp)
	for _, hit := range resp["hits"].(map[string]interface{})["hits"].([]map[string]interface{}) {
		r := hit["_source"].(map[string]interface{})
		rec := GroupRecord{
			Username:    r["username"].(string),
			Type:        r["type"].(string),
			Title:       r["title"].(string),
			MemberCount: r["member_count"].(int),
		}
		groups = append(groups, rec)
	}
	return groups
}

func init() {
	server := os.Getenv("OPENSEARCH_SERVER")
	if server == "" {
		log.Fatalln("environment OPENSEARCH_SERVER empty!")
	}

	var err error
	// Initialize the client with SSL/TLS enabled.
	opensvc, err = opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{server},
	})
	if err != nil {
		log.Panic("cannot initialize", err)
	}
}
