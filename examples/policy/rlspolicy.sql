\c userdb;
CREATE POLICY p1 ON t1 FOR ALL TO PUBLIC USING (a % 2 = 0); -- be even number
CREATE POLICY p2 ON t2 FOR ALL TO PUBLIC USING (a % 2 = 1); -- be odd number
