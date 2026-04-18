// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package business

import (
	"context"
	"fmt"

	"buf.build/gen/go/antinvestor/files/connectrpc/go/files/v1/filesv1connect"
	filesv1 "buf.build/gen/go/antinvestor/files/protocolbuffers/go/files/v1"
)

// uploadChunkSize is the maximum chunk size for streaming file uploads (64KB).
const uploadChunkSize = 64 * 1024

// NewFileUploadFunc creates a file upload function using the files service client.
// The returned function uploads a file via client-streaming RPC and returns the MXC URI.
func NewFileUploadFunc(
	client filesv1connect.FilesServiceClient,
) func(filename, contentType string, data []byte) (string, error) {
	return func(filename, contentType string, data []byte) (string, error) {
		ctx := context.Background()
		stream := client.UploadContent(ctx)

		// Send metadata as the first message.
		totalSize := int64(len(data))
		if err := stream.Send(&filesv1.UploadContentRequest{
			Data: &filesv1.UploadContentRequest_Metadata{
				Metadata: &filesv1.UploadMetadata{
					ContentType: contentType,
					Filename:    filename,
					TotalSize:   totalSize,
				},
			},
		}); err != nil {
			return "", fmt.Errorf("send upload metadata: %w", err)
		}

		// Send data in chunks.
		for offset := 0; offset < len(data); offset += uploadChunkSize {
			end := offset + uploadChunkSize
			if end > len(data) {
				end = len(data)
			}

			if err := stream.Send(&filesv1.UploadContentRequest{
				Data: &filesv1.UploadContentRequest_Chunk{
					Chunk: data[offset:end],
				},
			}); err != nil {
				return "", fmt.Errorf("send upload chunk: %w", err)
			}
		}

		// Close and receive response.
		resp, err := stream.CloseAndReceive()
		if err != nil {
			return "", fmt.Errorf("close upload stream: %w", err)
		}

		return resp.Msg.GetContentUri(), nil
	}
}
