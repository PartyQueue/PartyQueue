DROP TABLE IF EXISTS `requests`;
DROP TABLE IF EXISTS `songs`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `rooms`;

CREATE TABLE `rooms` (
    `id` varchar(6) NOT NULL,
    `host_token` varchar(100) NOT NULL,
    `created` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `last_used` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
);

CREATE TABLE `users` (
    `id` varchar(36) NOT NULL,
    `name` varchar(30),
    `first_seen` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `last_seen` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
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
    `duration` INT NOT NULL,
    `image` varchar(100),
    `last_req` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `popularity` INT DEFAULT 1,
    PRIMARY KEY (`uri`)
);

CREATE TABLE `requests` (
    `uri` varchar(30) NOT NULL,
    `user_id` varchar(36) NOT NULL,
    `room_id` varchar(6) NOT NULL,
    `priority` INT DEFAULT 1,
    `time` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
