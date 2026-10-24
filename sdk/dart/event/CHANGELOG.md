## 0.2.0

- Regenerated with protoc_plugin 25.1 / connect-dart 2.0; requires protobuf ^6.1.0, connectrpc ^2.0.0, fixnum ^1.1.1 and Dart ^3.7. Well-known types now come from `package:protobuf`.
- Requires `antinvestor_api_common` ^2.0.0.
- Removed the vendored `google/protobuf` well-known-type copies and the empty `.pbserver.dart` files that protoc_plugin 25 no longer emits.
- The barrel no longer exports `common.v1` messages: this module's protos do not import `common/v1/common.proto`, so those were stale generated files.

## 0.1.0

- Generated Dart SDK from proto definitions

