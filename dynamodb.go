package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var dynsvc *dynamodb.Client

func ddbWriteUser(ctx context.Context, u UserRecord) {
	// write user info
	_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String("users"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberN{Value: strconv.FormatInt(u.ID, 10)},
		},
		ReturnValues:     types.ReturnValueUpdatedOld,
		UpdateExpression: aws.String("set username = :username, first_name = :first_name, last_name = :last_name, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username":   &types.AttributeValueMemberS{Value: u.Username},
			":first_name": &types.AttributeValueMemberS{Value: u.FirstName},
			":last_name":  &types.AttributeValueMemberS{Value: u.LastName},
			":created_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
			":update_at":  &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		},
	})
	if err != nil {
		log.Printf("record user %d error: %v\n", u.ID, err)
		return
	}
}

func ddbWriteGroup(ctx context.Context, s *CommandState) {
	// write group info
	updatedOldValues, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String("groups"),
		Key: map[string]types.AttributeValue{
			"username": &types.AttributeValueMemberS{Value: s.UserName},
		},
		ReturnValues:     types.ReturnValueUpdatedOld,
		UpdateExpression: aws.String("set title = :title, #t = :type, description = :desc, chat_id = :chat_id, member_count = :member_count, category = :category, tags = :tags, update_at = :update_at, created_at = if_not_exists(created_at, :created_at)"),
		ExpressionAttributeNames: map[string]string{
			"#t": "type",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":title":        &types.AttributeValueMemberS{Value: s.Title},
			":type":         &types.AttributeValueMemberS{Value: s.Type},
			":desc":         &types.AttributeValueMemberS{Value: s.Description},
			":chat_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(s.ID, 10)},
			":member_count": &types.AttributeValueMemberN{Value: strconv.Itoa(s.MemberCount)},
			":created_at":   &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
			":update_at":    &types.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
			":category":     &types.AttributeValueMemberS{Value: s.Category},
			":tags":         &types.AttributeValueMemberSS{Value: s.Tags},
		},
	})
	if err != nil {
		log.Printf("index %s error: %v\n", s.UserName, err)
		return
	}

	// Write tags(for search) info
	// In this step, we move the group from the old tags indexes to the new tags'
	oldTags := []string{}
	checkOld := map[string]bool{}
	toDelete := []string{}
	toAdd := []string{}
	_ = attributevalue.Unmarshal(updatedOldValues.Attributes["tags"], oldTags)
	for _, o := range oldTags {
		checkOld[o] = true
	}
	for _, t := range s.Tags {
		if _, ok := checkOld[t]; !ok { // if the new tag doesn't overlap with old tags, we need to index the group use the new tag
			toAdd = append(toAdd, t)
		} else { // if overlapped, mark the old tag to false for reservation
			checkOld[t] = false
		}
	}
	for t, v := range checkOld {
		if v == false { // we have marked the overlapped tags to false
			continue
		}
		toDelete = append(toDelete, t)
	}

	var wg sync.WaitGroup
	for _, t := range toAdd {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String("tags"),
				Key: map[string]types.AttributeValue{
					"tag": &types.AttributeValueMemberS{Value: tag},
				},
				UpdateExpression: aws.String("add groups :group"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":group": &types.AttributeValueMemberSS{Value: []string{s.UserName}},
				},
			})
			if err != nil {
				log.Printf("add tag index %s:%s error: %v\n", tag, s.UserName, err)
			}
		}(t)
	}
	for _, t := range toDelete {
		wg.Add(1)
		go func(tag string) {
			defer wg.Done()
			_, err := dynsvc.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String("tags"),
				Key: map[string]types.AttributeValue{
					"tag": &types.AttributeValueMemberS{Value: tag},
				},
				UpdateExpression: aws.String("delete groups :group"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":group": &types.AttributeValueMemberSS{Value: []string{s.UserName}},
				},
			})
			if err != nil {
				log.Printf("delete tags index %s:%s error: %v\n", tag, s.UserName, err)
			}
		}(t)
	}
	wg.Wait()

}

func init() {
	// Initialize dynamodb client
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v\n", err)
	}

	// Using the Config value, create the DynamoDB client
	dynsvc = dynamodb.NewFromConfig(cfg)
}
