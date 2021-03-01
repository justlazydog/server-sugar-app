ALTER TABLE `open_cloud`.`sugars`
ADD COLUMN `avg_growth_rate` DECIMAL(32,16) NULL DEFAULT 0 AFTER `account_out`,
ADD COLUMN `dat` VARCHAR(45) NULL DEFAULT '' AFTER `avg_growth_rate`;