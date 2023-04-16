package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	models "github.com/Portfolio-Advanced-software/BingeBuster-MovieService/models"
	mongodb "github.com/Portfolio-Advanced-software/BingeBuster-MovieService/mongodb"
	moviepb "github.com/Portfolio-Advanced-software/BingeBuster-MovieService/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MovieServiceServer struct {
	moviepb.UnimplementedMovieServiceServer
}

func (s *MovieServiceServer) CreateMovie(ctx context.Context, req *moviepb.CreateMovieReq) (*moviepb.CreateMovieRes, error) {
	// Essentially doing req.Movie to access the struct with a nil check
	movie := req.GetMovie()
	if movie == nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid movie")
	}
	// Now we have to convert this into a Movie type to convert into BSON
	data := models.Movie{
		// ID:    Empty, so it gets omitted and MongoDB generates a unique Object ID upon insertion.
		Title:       movie.GetTitle(),
		Description: movie.GetDescription(),
		ReleaseDate: movie.GetReleaseDate(),
		Director:    movie.GetDirector(),
		Genre:       movie.GetGenre(),
		Rating:      movie.GetRating(),
		Runtime:     movie.GetRuntime(),
		Poster:      movie.GetPoster(),
	}

	// Insert the data into the database, result contains the newly generated Object ID for the new document
	result, err := moviedb.InsertOne(mongoCtx, data)
	// check for potential errors
	if err != nil {
		// return internal gRPC error to be handled later
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	// add the id to movie, first cast the "generic type" (go doesn't have real generics yet) to an Object ID.
	oid := result.InsertedID.(primitive.ObjectID)
	// Convert the object id to it's string counterpart
	movie.Id = oid.Hex()
	// return the blog in a CreateMovieRes type
	return &moviepb.CreateMovieRes{Movie: movie}, nil
}

func (s *MovieServiceServer) ReadMovie(ctx context.Context, req *moviepb.ReadMovieReq) (*moviepb.ReadMovieRes, error) {
	// convert string id (from proto) to mongoDB ObjectId
	oid, err := primitive.ObjectIDFromHex(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}
	result := moviedb.FindOne(ctx, bson.M{"_id": oid})
	// Create an empty movie to write our decode result to
	data := models.Movie{}
	// decode and write to data
	if err := result.Decode(&data); err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find movie with Object Id %s: %v", req.GetId(), err))
	}
	// Cast to ReadMovieRes type
	response := &moviepb.ReadMovieRes{
		Movie: &moviepb.Movie{
			Id:          oid.Hex(),
			Title:       data.Title,
			Description: data.Description,
			ReleaseDate: data.ReleaseDate,
			Director:    data.Director,
			Genre:       data.Genre,
			Rating:      data.Rating,
			Runtime:     data.Runtime,
			Poster:      data.Poster,
		},
	}
	return response, nil
}

func (s *MovieServiceServer) ListMovies(req *moviepb.ListMoviesReq, stream moviepb.MovieService_ListMoviesServer) error {
	// Initiate a movie type to write decoded data to
	data := &models.Movie{}
	// collection.Find returns a cursor for our (empty) query
	cursor, err := moviedb.Find(context.Background(), bson.M{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown internal error: %v", err))
	}
	// An expression with defer will be called at the end of the function
	defer cursor.Close(context.Background())
	// cursor.Next() returns a boolean, if false there are no more items and loop will break
	for cursor.Next(context.Background()) {
		// Decode the data at the current pointer and write it to data
		err := cursor.Decode(data)
		// check error
		if err != nil {
			return status.Errorf(codes.Unavailable, fmt.Sprintf("Could not decode data: %v", err))
		}
		// If no error is found send blog over stream
		stream.Send(&moviepb.ListMoviesRes{
			Movie: &moviepb.Movie{
				Id:          data.ID.Hex(),
				Title:       data.Title,
				Description: data.Description,
				ReleaseDate: data.ReleaseDate,
				Director:    data.Director,
				Genre:       data.Genre,
				Rating:      data.Rating,
				Runtime:     data.Runtime,
				Poster:      data.Poster,
			},
		})
	}
	// Check if the cursor has any errors
	if err := cursor.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unkown cursor error: %v", err))
	}
	return nil
}

func (s *MovieServiceServer) UpdateMovie(ctx context.Context, req *moviepb.UpdateMovieReq) (*moviepb.UpdateMovieRes, error) {
	// Get the movie data from the request
	movie := req.GetMovie()

	// Convert the Id string to a MongoDB ObjectId
	oid, err := primitive.ObjectIDFromHex(movie.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Could not convert the supplied movie id to a MongoDB ObjectId: %v", err),
		)
	}

	// Convert the data to be updated into an unordered Bson document
	update := bson.M{
		"title":       movie.GetTitle(),
		"description": movie.GetDescription(),
		"releaseDate": movie.GetReleaseDate(),
		"director":    movie.GetDirector(),
		"genre":       movie.GetGenre(),
		"rating":      movie.GetRating(),
		"runtime":     movie.GetRuntime(),
		"poster":      movie.GetPoster(),
	}

	// Convert the oid into an unordered bson document to search by id
	filter := bson.M{"_id": oid}

	// Result is the BSON encoded result
	// To return the updated document instead of original we have to add options.
	result := moviedb.FindOneAndUpdate(ctx, filter, bson.M{"$set": update}, options.FindOneAndUpdate().SetReturnDocument(1))

	// Decode result and write it to 'decoded'
	decoded := models.Movie{}
	err = result.Decode(&decoded)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Could not find movie with supplied ID: %v", err),
		)
	}
	return &moviepb.UpdateMovieRes{
		Movie: &moviepb.Movie{
			Id:          decoded.ID.Hex(),
			Title:       decoded.Title,
			Description: decoded.Description,
			ReleaseDate: decoded.ReleaseDate,
			Director:    decoded.Director,
			Genre:       decoded.Genre,
			Rating:      decoded.Rating,
			Runtime:     decoded.Runtime,
			Poster:      decoded.Poster,
		},
	}, nil
}

func (s *MovieServiceServer) DeleteMovie(ctx context.Context, req *moviepb.DeleteMovieReq) (*moviepb.DeleteMovieRes, error) {
	// Get the ID (string) from the request message and convert it to an Object ID
	oid, err := primitive.ObjectIDFromHex(req.GetId())
	// Check for errors
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Could not convert to ObjectId: %v", err))
	}
	// DeleteOne returns DeleteResult which is a struct containing the amount of deleted docs (in this case only 1 always)
	// So we return a boolean instead
	_, err = moviedb.DeleteOne(ctx, bson.M{"_id": oid})
	// Check for errors
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("Could not find/delete movie with id %s: %v", req.GetId(), err))
	}
	// Return response with success: true if no error is thrown (and thus document is removed)
	return &moviepb.DeleteMovieRes{
		Success: true,
	}, nil
}

const (
	port = ":50051"
)

var db *mongo.Client
var moviedb *mongo.Collection
var mongoCtx context.Context

var mongoUsername = "user-service"
var mongoPwd = "vLxxhmS0eJFwmteF"
var connUri = "mongodb+srv://" + mongoUsername + ":" + mongoPwd + "@cluster0.fpedw5d.mongodb.net/test"

var dbName = "MovieService"
var collectionName = "Movies"

func main() {
	// Configure 'log' package to give file name and line number on eg. log.Fatal
	// Pipe flags to one another (log.LstdFLags = log.Ldate | log.Ltime)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("Starting server on port :50051...")

	// Set listener to start server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Unable to listen on port %p: %v", lis.Addr(), err)
	}

	// Set options, here we can configure things like TLS support
	opts := []grpc.ServerOption{}
	// Create new gRPC server with (blank) options
	s := grpc.NewServer(opts...)
	// Create MovieService type
	srv := &MovieServiceServer{}

	// Register the service with the server
	moviepb.RegisterMovieServiceServer(s, srv)

	// Initialize MongoDb client
	fmt.Println("Connecting to MongoDB...")
	db = mongodb.ConnectToMongoDB(connUri)

	// Bind our collection to our global variable for use in other methods
	moviedb = db.Database(dbName).Collection(collectionName)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	fmt.Println("Server succesfully started on port :50051")

	// Right way to stop the server using a SHUTDOWN HOOK
	// Create a channel to receive OS signals
	c := make(chan os.Signal)

	// Relay os.Interrupt to our channel (os.Interrupt = CTRL+C)
	// Ignore other incoming signals
	signal.Notify(c, os.Interrupt)

	// Block main routine until a signal is received
	// As long as user doesn't press CTRL+C a message is not passed and our main routine keeps running
	<-c

	// After receiving CTRL+C Properly stop the server
	fmt.Println("\nStopping the server...")
	s.Stop()
	lis.Close()
	fmt.Println("Closing MongoDB connection")
	db.Disconnect(mongoCtx)
	fmt.Println("Done.")

}
