package social_message

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"koding/remoteapi/models"
)

// PostRemoteAPISocialMessageReplyReader is a Reader for the PostRemoteAPISocialMessageReply structure.
type PostRemoteAPISocialMessageReplyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PostRemoteAPISocialMessageReplyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewPostRemoteAPISocialMessageReplyOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewPostRemoteAPISocialMessageReplyUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewPostRemoteAPISocialMessageReplyOK creates a PostRemoteAPISocialMessageReplyOK with default headers values
func NewPostRemoteAPISocialMessageReplyOK() *PostRemoteAPISocialMessageReplyOK {
	return &PostRemoteAPISocialMessageReplyOK{}
}

/*PostRemoteAPISocialMessageReplyOK handles this case with default header values.

Request processed successfully
*/
type PostRemoteAPISocialMessageReplyOK struct {
	Payload *models.DefaultResponse
}

func (o *PostRemoteAPISocialMessageReplyOK) Error() string {
	return fmt.Sprintf("[POST /remote.api/SocialMessage.reply][%d] postRemoteApiSocialMessageReplyOK  %+v", 200, o.Payload)
}

func (o *PostRemoteAPISocialMessageReplyOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.DefaultResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPostRemoteAPISocialMessageReplyUnauthorized creates a PostRemoteAPISocialMessageReplyUnauthorized with default headers values
func NewPostRemoteAPISocialMessageReplyUnauthorized() *PostRemoteAPISocialMessageReplyUnauthorized {
	return &PostRemoteAPISocialMessageReplyUnauthorized{}
}

/*PostRemoteAPISocialMessageReplyUnauthorized handles this case with default header values.

Unauthorized request
*/
type PostRemoteAPISocialMessageReplyUnauthorized struct {
	Payload *models.UnauthorizedRequest
}

func (o *PostRemoteAPISocialMessageReplyUnauthorized) Error() string {
	return fmt.Sprintf("[POST /remote.api/SocialMessage.reply][%d] postRemoteApiSocialMessageReplyUnauthorized  %+v", 401, o.Payload)
}

func (o *PostRemoteAPISocialMessageReplyUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.UnauthorizedRequest)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
