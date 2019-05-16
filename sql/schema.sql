DROP TABLE IF EXISTS `requests`;
DROP TABLE IF EXISTS `songs`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `rooms`;

CREATE TABLE `rooms` (
    `id` varchar(6) NOT NULL,
    `host_token` varchar(100) NOT NULL,
    `created` datetime(6),
    `last_used` datetime(6),
    `started_at` datetime(6),
    `paused_at` datetime(6),
    `current_duration` bigint(20),
    PRIMARY KEY (`id`)
);

CREATE TABLE `users` (
    `id` varchar(36) NOT NULL,
    `name` varchar(30),
    `first_seen` datetime(6),
    `last_seen` datetime(6),
    `host_room_id` varchar(6) NULL,
    `guest_room_id` varchar(6) NULL,
    PRIMARY KEY (`id`),
    CONSTRAINT fk_user_host_room_id
        FOREIGN KEY (`host_room_id`)
        REFERENCES rooms(`id`)
        ON DELETE SET NULL,
    CONSTRAINT fk_user_guest_room_id
        FOREIGN KEY (`guest_room_id`)
        REFERENCES rooms(`id`)
        ON DELETE SET NULL
);

CREATE TABLE `songs` (
    `uri` varchar(30) NOT NULL,
    `artist_uris` varchar(120) NOT NULL,
    `title` varchar(50) NOT NULL,
    `artist` varchar(50) NOT NULL,
    `duration` bigint(20) NOT NULL,
    `image` varchar(100),
    `last_req` datetime(6),
    `popularity` bigint(20) DEFAULT 1,
    PRIMARY KEY (`uri`)
);

CREATE TABLE `requests` (
    `uri` varchar(30) NOT NULL,
    `user_id` varchar(36) NOT NULL,
    `room_id` varchar(6) NOT NULL,
    `priority` bigint(20) DEFAULT 1,
    `time` datetime(6),
    PRIMARY KEY (`uri`, `user_id`, `room_id`),
    CONSTRAINT fk_req_song_uri
        FOREIGN KEY (`uri`)
        REFERENCES songs(`uri`)
        ON DELETE CASCADE,
    CONSTRAINT fk_req_user_id
        FOREIGN KEY (`user_id`)
        REFERENCES users(`id`)
        ON DELETE CASCADE,
    CONSTRAINT fk_req_room_id
        FOREIGN KEY (`room_id`)
        REFERENCES rooms(`id`)
        ON DELETE CASCADE
);
