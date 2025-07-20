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
