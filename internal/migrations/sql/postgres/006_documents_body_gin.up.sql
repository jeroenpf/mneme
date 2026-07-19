CREATE INDEX documents_body_gin ON documents USING GIN (body);
