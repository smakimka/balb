CREATE USER server with password 'server_password';
CREATE DATABASE server_db;
GRANT ALL PRIVILEGES ON DATABASE server_db TO server;
ALTER DATABASE server_db OWNER TO server;

CREATE USER bot with password 'bot_password';
CREATE DATABASE bot_db;
GRANT ALL PRIVILEGES ON DATABASE bot_db TO bot;
ALTER DATABASE bot_db OWNER TO bot;
