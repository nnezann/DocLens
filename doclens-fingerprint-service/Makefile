.PHONY: proto test

proto:
	python -m grpc_tools.protoc -I proto --python_out=src --grpc_python_out=src proto/doclens/fingerprint/v1/fingerprint.proto

test:
	PYTHONPATH=src python -m unittest discover tests/ -v
