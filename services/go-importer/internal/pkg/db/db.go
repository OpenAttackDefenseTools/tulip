package db

import (
	"context"
	"encoding/base64"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Added a flow struct
type FlowItem struct {
	/// From: "s" / "c" for server or client
	From string
	/// Data, in a somewhat readable format
	Data string
	/// The raw data, base64 encoded.
	// TODO; Replace this with gridfs
	B64 string
	/// Timestamp of the first packet in the flow (Epoch / ms)
	Time int
}

type FlowEntry struct {
	Src_port     int
	Dst_port     int
	Src_ip       string
	Dst_ip       string
	Time         int
	Duration     int
	Num_packets  int
	Blocked      bool
	Filename     string
	Parent_id    primitive.ObjectID
	Child_id     primitive.ObjectID
	Fingerprints []uint32
	Suricata     []int
	Flow         []FlowItem
	Tags         []string
	Size         int
}

type Database struct {
	client *mongo.Client
}

func ConnectMongo(uri string) Database {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}

	return Database{
		client: client,
	}
}

func (db Database) ConfigureDatabase() {
	db.InsertTag("flag-in")
	db.InsertTag("flag-out")
	db.InsertTag("blocked")
	db.InsertTag("suricata")
	db.InsertTag("starred")
	db.InsertTag("tcp")
	db.InsertTag("udp")
	db.ConfigureIndexes()
}

func (db Database) ConfigureIndexes() {
	// create Index
	flowCollection := db.client.Database("pcap").Collection("pcap")

	_, err := flowCollection.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		// time index (range filtering)
		{
			Keys: bson.D{
				{"time", 1},
			},
		},
		// data index (context filtering)
		{
			Keys: bson.D{
				{"data", "text"},
			},
		},
		// port combo index (traffic correlation)
		{
			Keys: bson.D{
				{"src_port", 1},
				{"dst_port", 1},
			},
		},
	})

	if err != nil {
		panic(err)
	}
}

// Flows are either coming from a file, in which case we'll dedupe them by pcap file name.
// If they're coming from a live capture, we can do pretty much the same thing, but we'll
// just have to come up with a label. (e.g. capture-<epoch>)
// We can always swap this out with something better, but this is how flower currently handles deduping.
//
// A single flow is defined by a db.FlowEntry" struct, containing an array of flowitems and some metadata
func (db Database) InsertFlow(flow FlowEntry) {
	flowCollection := db.client.Database("pcap").Collection("pcap")

	// Process the data, so it works well in mongodb
	for idx := 0; idx < len(flow.Flow); idx++ {
		flowItem := &flow.Flow[idx]
		// Base64 encode the raw data string
		flowItem.B64 = base64.StdEncoding.EncodeToString([]byte(flowItem.Data))
		// filter the data string down to only printable characters
		flowItem.Data = strings.Map(func(r rune) rune {
			if r < 128 {
				return r
			}
			return -1
		}, flowItem.Data)
	}

	if len(flow.Fingerprints) > 0 {
		query := bson.M{
			"fingerprints": bson.M{
				"$in": flow.Fingerprints,
			},
		}
		opts := options.FindOne().SetSort(bson.M{"time": -1})

		// TODO does this return the first one? If multiple documents satisfy the given query expression, then this method will return the first document according to the natural order which reflects the order of documents on the disk.
		connectedFlow := struct {
			MongoID primitive.ObjectID `bson:"_id"`
		}{}
		err := flowCollection.FindOne(context.TODO(), query, opts).Decode(&connectedFlow)

		// There is a connected flow
		if err == nil {
			//TODO Maybe add the childs fingerprints to mine?
			flow.Child_id = connectedFlow.MongoID
		}
	}

	// TODO; use insertMany instead
	insertion, err := flowCollection.InsertOne(context.TODO(), flow)
	// check for errors in the insertion
	if err != nil {
		log.Println("Error occured while inserting record: ", err)
		log.Println("NO PCAP DATA WILL BE AVAILABLE FOR: ", flow.Filename)
	}

	if flow.Child_id == primitive.NilObjectID {
		return
	}

	query := bson.M{
		"_id": flow.Child_id,
	}

	info := bson.M{
		"$set": bson.M{
			"parent_id": insertion.InsertedID,
		},
	}

	_, err = flowCollection.UpdateOne(context.TODO(), query, info)
	//TODO error handling
}

type PcapFile struct {
	FileName string `bson:"file_name"`
	Position int64 `bson:"position"`
}

// Insert a new pcap uri, returns true if the pcap was not present yet,
// otherwise returns false
func (db Database) InsertPcap(uri string, position int64) bool {
	files := db.client.Database("pcap").Collection("filesImported")
	exists, _ := db.GetPcap(uri)
	if !exists {
		files.InsertOne(context.TODO(), bson.M{"file_name": uri,"position": position})
	} else {
		files.UpdateOne(context.TODO(), bson.M{"file_name": uri}, bson.M{"$set":bson.M{"position": position}})
	}
	return !exists
}

func (db Database) GetPcap(uri string) (bool, PcapFile) {
	files := db.client.Database("pcap").Collection("filesImported")
	var result PcapFile
	match := files.FindOne(context.TODO(), bson.M{"file_name": uri})
	match.Decode(&result)
	return match.Err() != mongo.ErrNoDocuments, result
}

type FlowID struct {
	Src_port int
	Dst_port int
	Src_ip   string
	Dst_ip   string
	Time     time.Time
}

type Signature struct {
	MongoID primitive.ObjectID `bson:"_id,omitempty"`
	ID      int
	Msg     string
	Action  string
	Tag     string `bson:"omitempty"`
}

func (db Database) AddSignature(sig Signature) string {
	sigCollection := db.client.Database("pcap").Collection("signatures")

	// TODO; there's a bit of a race here, but I'm also racing to get this code working in time
	// for the next demo, so it all evens out.

	query := bson.M{
		"id":     sig.ID,
		"msg":    sig.Msg,
		"action": sig.Action,
		"tag":    sig.Tag,
	}

	var existing_sig Signature
	err := sigCollection.FindOne(context.TODO(), query).Decode(&existing_sig)
	if err != nil {
		// The signature does not appear in the DB yet. Let's add it.
		res, err := sigCollection.InsertOne(context.TODO(), query)
		if err != nil {
			log.Println("Rule add failed with error: ", err)
			return ""
		}
		ret := res.InsertedID.(primitive.ObjectID)
		return ret.Hex()
	} else {
		// The signature _does_ appear in the db. Let's return it's ID directly!
		return existing_sig.MongoID.Hex()
	}
}

func (db Database) findFlowInDB(flow FlowID, window int) (mongo.Collection, bson.M) {
	// Find a flow that more or less matches the one we're looking for
	flowCollection := db.client.Database("pcap").Collection("pcap")
	epoch := int(flow.Time.UnixNano() / 1000000)
	filter := bson.M{
		"src_port": flow.Src_port,
		"dst_port": flow.Dst_port,
		"src_ip":   flow.Src_ip,
		"dst_ip":   flow.Dst_ip,
		"time": bson.M{
			"$gt": epoch - window,
			"$lt": epoch + window,
		},
	}

	return *flowCollection, filter
}

func (db Database) updateFlowInDB(flowCollection mongo.Collection, filter bson.M, update bson.M) bool {
	// Enrich the flow with tag information
	res, err := flowCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		log.Println("Error occured while editing record:", err)
		return false
	}

	return res.MatchedCount > 0
}

func (db Database) AddSignatureToFlow(flow FlowID, sig Signature, window int) bool {
	// Add the signature to the collection
	sig_id := db.AddSignature(sig)
	if sig_id == "" {
		return false
	}

	tags := []string{"suricata"}
	flowCollection, filter := db.findFlowInDB(flow, window)

	// Add tag from the signature if it contained one
	if sig.Tag != "" {
		db.InsertTag(sig.Tag)
		tags = append(tags, sig.Tag)
	}

	var update bson.M
	// TODO; This can probably be done more elegantly, right?
	if sig.Action == "blocked" {
		update = bson.M{
			"$set": bson.M{
				"blocked": true,
			},
			"$addToSet": bson.M{
				"tags": bson.M{
					"$each": append(tags, "blocked"),
				},
				"suricata": sig_id,
			},
		}
	} else {
		update = bson.M{
			"$addToSet": bson.M{
				"tags": bson.M{
					"$each": tags,
				},
				"suricata": sig_id,
			},
		}
	}

	return db.updateFlowInDB(flowCollection, filter, update)
}

func (db Database) AddTagsToFlow(flow FlowID, tags []string, window int) bool {
	flowCollection, filter := db.findFlowInDB(flow, window)

	// Add tags to tag collection
	for _, tag := range tags {
		db.InsertTag(tag)
	}

	// Update this flow with the tags
	update := bson.M{
		"$addToSet": bson.M{
			"tags": bson.M{
				"$each": tags,
			},
		},
	}

	// Apply update to database
	return db.updateFlowInDB(flowCollection, filter, update)

}
func (db Database) InsertTag(tag string) {
	tagCollection := db.client.Database("pcap").Collection("tags")
	// Yeah this will err... A lot.... Two more dev days till Athens, this will do.
	tagCollection.InsertOne(context.TODO(), bson.M{"_id": tag})
}
