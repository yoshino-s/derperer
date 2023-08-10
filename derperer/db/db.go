package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"tailscale.com/tailcfg"
)

type DB struct {
	client *mongo.Client
	ctx    context.Context
}

func New(ctx context.Context, uri string) (*DB, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &DB{
		client: client,
	}, nil
}

func (db *DB) Disconnect() error {
	return db.client.Disconnect(db.ctx)
}

func (db *DB) getDERPRegions() ([]tailcfg.DERPRegion, error) {
	collection := db.client.Database("derperer").Collection("regions")
	cursor, err := collection.Find(db.ctx, bson.M{
		"$or": []bson.M{
			{"banned": bson.M{"$exists": false}},
			{"banned": false},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(db.ctx)

	var regions []tailcfg.DERPRegion
	for cursor.Next(db.ctx) {
		var region tailcfg.DERPRegion
		if err := cursor.Decode(&region); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}
	return regions, nil
}

func (db *DB) GetNextRegionID() (int, error) {
	regionID := db.client.Database("derperer").Collection("regionID")
	// start from 114000
	res, err := regionID.UpdateOne(db.ctx, bson.M{"regionid": bson.M{"$exists": true}}, bson.M{"$inc": bson.M{"regionid": 1}})
	if err != nil {
		return 0, err
	}
	if res.ModifiedCount == 0 {
		_, err := regionID.InsertOne(db.ctx, bson.M{"regionid": 114000})
		if err != nil {
			return 0, err
		}
		return 114000, nil
	}
	var region struct {
		RegionID int `bson:"regionid"`
	}
	err = regionID.FindOne(db.ctx, bson.M{}).Decode(&region)
	if err != nil {
		return 0, err
	}
	return region.RegionID, nil
}

func (db *DB) GetDERPMap() (*tailcfg.DERPMap, error) {
	derpMap := &tailcfg.DERPMap{
		Regions: map[int]*tailcfg.DERPRegion{},
	}

	regions, err := db.getDERPRegions()
	if err != nil {
		return derpMap, err
	}

	for idx, region := range regions {
		derpMap.Regions[region.RegionID] = &regions[idx]
	}
	return derpMap, nil
}

func (db *DB) Drop() error {
	err := db.client.Database("derperer").Collection("regions").Drop(db.ctx)
	if err != nil {
		return err
	}
	err = db.client.Database("derperer").Collection("regionID").Drop(db.ctx)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) BanRegion(regionID int) (int, error) {
	collection := db.client.Database("derperer").Collection("regions")
	res, err := collection.UpdateOne(db.ctx, bson.M{"regionid": regionID}, bson.M{"$set": bson.M{"banned": true}})
	return int(res.ModifiedCount), err
}

func (db *DB) InsertDERPRegion(region *tailcfg.DERPRegion) error {
	collection := db.client.Database("derperer").Collection("regions")
	// find region with same name
	var result tailcfg.DERPRegion
	err := collection.FindOne(db.ctx, bson.M{"regionname": region.RegionName}).Decode(&result)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			return err
		} else {
			regionId, err := db.GetNextRegionID()
			if err != nil {
				return err
			}
			region.RegionID = regionId
			for _, node := range region.Nodes {
				node.RegionID = regionId
			}
			_, err = collection.InsertOne(db.ctx, region)
			return err
		}
	} else {
		var nodeMap = map[string]*tailcfg.DERPNode{}
		for _, node := range region.Nodes {
			node.RegionID = result.RegionID
			nodeMap[node.Name] = node
		}

		for _, node := range result.Nodes {
			if _, ok := nodeMap[node.Name]; !ok {
				region.Nodes = append(region.Nodes, node)
			}
		}

		var nodes []*tailcfg.DERPNode
		nodes = append(nodes, region.Nodes...)

		_, err = collection.UpdateOne(db.ctx, bson.M{"regionname": region.RegionName}, bson.M{"$set": bson.M{"nodes": nodes}})

		return err
	}
}
