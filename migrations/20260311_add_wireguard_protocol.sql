ALTER TABLE nodes DROP CONSTRAINT IF EXISTS nodes_protocol_check;

ALTER TABLE nodes
ADD CONSTRAINT nodes_protocol_check
CHECK (protocol IN ('trojan', 'vmess', 'vless', 'ss', 'wireguard', 'anytls', 'hysteria2', 'tuic'));

COMMENT ON COLUMN nodes.protocol IS '协议类型：trojan, vmess, vless, ss, wireguard, anytls, hysteria2, tuic';
