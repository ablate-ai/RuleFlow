ALTER TABLE nodes DROP CONSTRAINT IF EXISTS nodes_protocol_check;

ALTER TABLE nodes
    ADD CONSTRAINT nodes_protocol_check
    CHECK (protocol IN ('trojan', 'vmess', 'vless', 'ss', 'anytls', 'hysteria2', 'tuic'));
