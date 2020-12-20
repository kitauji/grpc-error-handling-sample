package main

import (
	pb "kitauji/greeter"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetGrpcError(code codes.Code, msg string) error {
	return status.Errorf(code, msg)
}

func GetGrpcBadRequestError(code codes.Code, msg string, desc string) error {
	st := status.New(code, msg)
	details := &errdetails.BadRequest{
		FieldViolations: []*errdetails.BadRequest_FieldViolation{
			{Field: "Name", Description: desc},
		},
	}

	stDetails, _ := st.WithDetails(details)
	return stDetails.Err()
}

func GetGrpcCustomError(code codes.Code, msg string, errorno int32, desc string) error {
	st := status.New(code, msg)
	customErr := &pb.CustomError{
		ErrorNo:     errorno,
		Description: desc,
	}

	stDetails, _ := st.WithDetails(customErr)
	return stDetails.Err()
}
