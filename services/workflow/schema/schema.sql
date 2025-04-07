-- PostgreSQL schema for Workflow Service
-- Stores workflows as graphs of nodes with relationships
-- Generated based on workflow_node.proto and service documentation

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    description TEXT,
    status INT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    status INT,            -- protobuf: Status enum
    node BYTEA,            -- protobuf: Node, all fields except these...
    all_tasks BYTEA,       -- protobuf: repeated Task messages (binary blob)
    edits BYTEA,           -- protobuf: repeated NodeEdit messages (binary blob)
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_nodes_workflow_id ON nodes(workflow_id);
CREATE INDEX idx_nodes_node_id ON nodes(id);

-- Explicit graph edges (parent-child relationships)
CREATE TABLE node_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    parent_node_id TEXT NOT NULL,
    child_node_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_node_edges_workflow_id ON node_edges(workflow_id);
CREATE INDEX idx_node_edges_parent_child ON node_edges(parent_node_id, child_node_id);
