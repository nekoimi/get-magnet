SET NAMES utf8mb4;
SET
FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for magnets
-- ----------------------------
DROP TABLE IF EXISTS `magnets`;
CREATE TABLE `magnets`
(
    `id`           int      NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `created_at`   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录创建时间',
    `updated_at`   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录更新时间',
    `title`        varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL COMMENT '磁力资源标题',
    `number`       varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL COMMENT '磁力资源编号',
    `optimal_link` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL COMMENT '最优磁力链接',
    `links`        json NULL COMMENT '全部磁力链接',
    `res_host`     varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL COMMENT '资源Host',
    `res_path`     varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL COMMENT '资源路径',
    `status`       tinyint UNSIGNED NOT NULL DEFAULT 0 COMMENT '资源状态',
    PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB AUTO_INCREMENT = 433 CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = Dynamic;

SET
FOREIGN_KEY_CHECKS = 1;