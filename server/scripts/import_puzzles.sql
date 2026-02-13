-- Скрипт импорта данных пазлов из PDF
-- Запуск: psql -d <database> -f import_puzzles.sql
-- или для SQLite: sqlite3 <database.db> < import_puzzles.sql

-- Обновляем названия пазлов
-- Лампочка (пазлы 1-5)
UPDATE puzzles SET name = 'Лампочка (черный)' WHERE id = 1;
UPDATE puzzles SET name = 'Лампочка (зеленый)' WHERE id = 2;
UPDATE puzzles SET name = 'Лампочка (красный)' WHERE id = 3;
UPDATE puzzles SET name = 'Лампочка (цвет 4)' WHERE id = 4;
UPDATE puzzles SET name = 'Лампочка (цвет 5)' WHERE id = 5;

-- Золотое сечение (пазлы 6-10)
UPDATE puzzles SET name = 'Золотое сечение (черный)' WHERE id = 6;
UPDATE puzzles SET name = 'Золотое сечение (зеленый)' WHERE id = 7;
UPDATE puzzles SET name = 'Золотое сечение (красный)' WHERE id = 8;
UPDATE puzzles SET name = 'Золотое сечение (цвет 4)' WHERE id = 9;
UPDATE puzzles SET name = 'Золотое сечение (цвет 5)' WHERE id = 10;

-- Атом (пазлы 11-15)
UPDATE puzzles SET name = 'Атом (черный)' WHERE id = 11;
UPDATE puzzles SET name = 'Атом (зеленый)' WHERE id = 12;
UPDATE puzzles SET name = 'Атом (красный)' WHERE id = 13;
UPDATE puzzles SET name = 'Атом (цвет 4)' WHERE id = 14;
UPDATE puzzles SET name = 'Атом (цвет 5)' WHERE id = 15;

-- Шестерни (пазлы 16-20)
UPDATE puzzles SET name = 'Шестерни (черный)' WHERE id = 16;
UPDATE puzzles SET name = 'Шестерни (зеленый)' WHERE id = 17;
UPDATE puzzles SET name = 'Шестерни (красный)' WHERE id = 18;
UPDATE puzzles SET name = 'Шестерни (цвет 4)' WHERE id = 19;
UPDATE puzzles SET name = 'Шестерни (цвет 5)' WHERE id = 20;

-- Бесконечность (пазлы 21-25)
UPDATE puzzles SET name = 'Бесконечность (черный)' WHERE id = 21;
UPDATE puzzles SET name = 'Бесконечность (зеленый)' WHERE id = 22;
UPDATE puzzles SET name = 'Бесконечность (красный)' WHERE id = 23;
UPDATE puzzles SET name = 'Бесконечность (цвет 4)' WHERE id = 24;
UPDATE puzzles SET name = 'Бесконечность (цвет 5)' WHERE id = 25;

-- Осциллограф (пазлы 26-30)
UPDATE puzzles SET name = 'Осциллограф (черный)' WHERE id = 26;
UPDATE puzzles SET name = 'Осциллограф (зеленый)' WHERE id = 27;
UPDATE puzzles SET name = 'Осциллограф (красный)' WHERE id = 28;
UPDATE puzzles SET name = 'Осциллограф (цвет 4)' WHERE id = 29;
UPDATE puzzles SET name = 'Осциллограф (цвет 5)' WHERE id = 30;

-- Удаляем старые детали (если есть)
DELETE FROM puzzle_pieces;

-- Вставляем детали пазлов
-- Лампочка (черный) - пазл 1
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('49HGX3L', 1, 1), ('ZHY3EZX', 1, 2), ('9T5CPDJ', 1, 3),
('Y9ORIL2', 1, 4), ('HFK5LO5', 1, 5), ('NR4R5B6', 1, 6);

-- Лампочка (зеленый) - пазл 2
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('U58QKZL', 2, 1), ('NUOYRF0', 2, 2), ('8KFAYR8', 2, 3),
('WAECTQS', 2, 4), ('DBKFN7Z', 2, 5), ('4TFE1EH', 2, 6);

-- Лампочка (красный) - пазл 3
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('PIUWWC8', 3, 1), ('U3XMWH8', 3, 2), ('341PG51', 3, 3),
('M85XYLZ', 3, 4), ('GLBQ6LT', 3, 5), ('OJ34HLF', 3, 6);

-- Лампочка (цвет 4) - пазл 4
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('STLWMXO', 4, 1), ('WFVV5IK', 4, 2), ('858R9CK', 4, 3),
('U1UKEM5', 4, 4), ('7GFR4FJ', 4, 5), ('YUV7NZ7', 4, 6);

-- Лампочка (цвет 5) - пазл 5
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('SVOMBPT', 5, 1), ('7VFFL6A', 5, 2), ('YDC5KKL', 5, 3),
('LSRU40B', 5, 4), ('RWI6FDE', 5, 5), ('7QTNB6U', 5, 6);

-- Золотое сечение (черный) - пазл 6
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('IXQV5E8', 6, 1), ('ONT91JT', 6, 2), ('S1S5WQN', 6, 3),
('L9F63V2', 6, 4), ('5STOHF5', 6, 5), ('YN1CD9P', 6, 6);

-- Золотое сечение (зеленый) - пазл 7
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('5G2A6WU', 7, 1), ('DWZSJDM', 7, 2), ('AP33ULC', 7, 3),
('LJVDOUT', 7, 4), ('5R5DA7V', 7, 5), ('4V06NOO', 7, 6);

-- Золотое сечение (красный) - пазл 8
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('34HSR0W', 8, 1), ('SAJAOK1', 8, 2), ('QIY5A70', 8, 3),
('LN9N0IT', 8, 4), ('F5I2L63', 8, 5), ('JSKVSHA', 8, 6);

-- Золотое сечение (цвет 4) - пазл 9
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('U22C99V', 9, 1), ('JY38UM7', 9, 2), ('HMSTAY0', 9, 3),
('26SQVBA', 9, 4), ('FJ1JJB8', 9, 5), ('60SAEVS', 9, 6);

-- Золотое сечение (цвет 5) - пазл 10
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('5QON8BN', 10, 1), ('UPDFGCG', 10, 2), ('FP8MPPS', 10, 3),
('QYOU4R8', 10, 4), ('WASYCMT', 10, 5), ('KH60D8S', 10, 6);

-- Атом (черный) - пазл 11
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('R861NID', 11, 1), ('DOVL42O', 11, 2), ('U7SF2UQ', 11, 3),
('6LZQNPQ', 11, 4), ('6HCC7DD', 11, 5), ('XQ0VB7F', 11, 6);

-- Атом (зеленый) - пазл 12
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('FS6TUVK', 12, 1), ('VH8J4M9', 12, 2), ('82C3R3V', 12, 3),
('KBB8YGL', 12, 4), ('TMO77P1', 12, 5), ('UW7SMSU', 12, 6);

-- Атом (красный) - пазл 13
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('DVHGFDZ', 13, 1), ('XXU4WAV', 13, 2), ('1EDZJLI', 13, 3),
('I39X60N', 13, 4), ('6UNRQ11', 13, 5), ('ONVNAX9', 13, 6);

-- Атом (цвет 4) - пазл 14
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('LQ9034E', 14, 1), ('LLAF6Q1', 14, 2), ('S2SXGPY', 14, 3),
('35082Z9', 14, 4), ('E18O6BO', 14, 5), ('ZQF81KV', 14, 6);

-- Атом (цвет 5) - пазл 15
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('UMANRN9', 15, 1), ('SX04LTQ', 15, 2), ('1OVI33K', 15, 3),
('GLY7S2U', 15, 4), ('3OCALA3', 15, 5), ('ZH9P58Y', 15, 6);

-- Шестерни (черный) - пазл 16
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('18A0USQ', 16, 1), ('5VPDPAP', 16, 2), ('0CPX98S', 16, 3),
('8Q9DFY1', 16, 4), ('ZUDX5XG', 16, 5), ('CB9Y2ZA', 16, 6);

-- Шестерни (зеленый) - пазл 17
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('H9K574G', 17, 1), ('ISVASUY', 17, 2), ('BIEKLXF', 17, 3),
('UV2BUOR', 17, 4), ('ANDPGGB', 17, 5), ('3E21S0W', 17, 6);

-- Шестерни (красный) - пазл 18
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('RFCHI4F', 18, 1), ('HDI8GKN', 18, 2), ('TM5RZXM', 18, 3),
('RT8Y1AK', 18, 4), ('VP9OBGG', 18, 5), ('YNLSMQX', 18, 6);

-- Шестерни (цвет 4) - пазл 19
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('T5OC99Q', 19, 1), ('YH9OJK9', 19, 2), ('1BTPE6B', 19, 3),
('OCA8Q4S', 19, 4), ('ZI4Q59M', 19, 5), ('6OFI7VD', 19, 6);

-- Шестерни (цвет 5) - пазл 20
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('C0JJBEM', 20, 1), ('XRC9BRG', 20, 2), ('XK58ONS', 20, 3),
('GK6G594', 20, 4), ('BKO4SCS', 20, 5), ('A76I952', 20, 6);

-- Бесконечность (черный) - пазл 21
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('E45JIYG', 21, 1), ('EI1X854', 21, 2), ('SNW4AFR', 21, 3),
('3103TVR', 21, 4), ('SYAOOQL', 21, 5), ('DBL5IBQ', 21, 6);

-- Бесконечность (зеленый) - пазл 22
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('778UCKW', 22, 1), ('J8DYGTF', 22, 2), ('YU5FQ78', 22, 3),
('5CN9PJW', 22, 4), ('CWKNJJL', 22, 5), ('1VHK47F', 22, 6);

-- Бесконечность (красный) - пазл 23
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('M8YI4WK', 23, 1), ('7S7HW7Y', 23, 2), ('2CEOLUE', 23, 3),
('3076ISM', 23, 4), ('E3WGV7J', 23, 5), ('2CN9QUJ', 23, 6);

-- Бесконечность (цвет 4) - пазл 24
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('MRLE5O3', 24, 1), ('9IU8EX4', 24, 2), ('P95PMDZ', 24, 3),
('VVR4Q7X', 24, 4), ('35CGU9F', 24, 5), ('LBILLY1', 24, 6);

-- Бесконечность (цвет 5) - пазл 25
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('VJBU6KX', 25, 1), ('JSV4GH2', 25, 2), ('PPT5OMZ', 25, 3),
('SLJZ9RZ', 25, 4), ('VHZYKU2', 25, 5), ('0D6DF51', 25, 6);

-- Осциллограф (черный) - пазл 26
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('FV6EZUJ', 26, 1), ('56BOI2I', 26, 2), ('LM1ERXF', 26, 3),
('5HTF1KV', 26, 4), ('ADXMRSA', 26, 5), ('YC375JO', 26, 6);

-- Осциллограф (зеленый) - пазл 27
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('7PZHZPT', 27, 1), ('912J7JZ', 27, 2), ('831EIKI', 27, 3),
('LPFFYCB', 27, 4), ('RU3AJW7', 27, 5), ('DC5BJYP', 27, 6);

-- Осциллограф (красный) - пазл 28
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('TWZSDYP', 28, 1), ('4KXTLI1', 28, 2), ('F2WQJIP', 28, 3),
('WPF1V0K', 28, 4), ('SWDSL9Z', 28, 5), ('H57VDWL', 28, 6);

-- Осциллограф (цвет 4) - пазл 29
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('JFAJO2A', 29, 1), ('BU3S8I1', 29, 2), ('58A5JVF', 29, 3),
('DX62BCL', 29, 4), ('1HEAUOZ', 29, 5), ('YV1GEVA', 29, 6);

-- Осциллограф (цвет 5) - пазл 30
INSERT INTO puzzle_pieces (code, puzzle_id, piece_number) VALUES
('I3XZIY8', 30, 1), ('NW7ZAK6', 30, 2), ('OMNI6O1', 30, 3),
('4JJJRAQ', 30, 4), ('DYM3CBZ', 30, 5), ('27U0Y3P', 30, 6);
