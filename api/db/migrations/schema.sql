create table users (
  id text primary key,
  email text not null unique,
  username text not null unique,
  name text not null,
  password text not null,
  created_at datetime default current_timestamp,
  updated_at datetime
);

create index if not exists idx_users_email on users(email);

create table tunnels (
  id text primary key,
  user_id text not null,
  name text not null unique,
  domain text not null unique,
  target text not null,
  protocol text not null,
  ipv4 text not null,
  created_at datetime default current_timestamp,
  updated_at datetime,
  foreign key (user_id) references users(id) on delete cascade
);


create table refresh_tokens (
  id text primary key,
  user_id text not null,
  foreign key (user_id) references users(id) on delete cascade
);

create index if not exists idx_refresh_tokens_id on refresh_tokens(id);
create index if not exists idx_refresh_tokens_user_id on refresh_tokens(user_id);

create table api_keys (
  id text primary key,
  user_id text not null,
  name text not null,
  key_hash text not null unique,
  role text not null,
  is_active boolean default true,
  created_at datetime default current_timestamp,
  expires_at datetime,
  updated_at datetime default null,
  foreign key (user_id) references users(id) on delete cascade
);

create index if not exists idx_api_keys_hash on api_keys(key_hash);
create index if not exists idx_api_keys_user_id on api_keys(user_id);
