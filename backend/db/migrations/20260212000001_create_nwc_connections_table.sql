-- migrate:up
CREATE TABLE nwc_connections (
    id serial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) UNIQUE,
    connection_uri text NOT NULL,
    expires_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);

-- migrate:down
DROP TABLE IF EXISTS nwc_connections;
