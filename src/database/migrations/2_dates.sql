UPDATE feeds SET failing_since = datetime(failing_since) WHERE failing_since IS NOT NULL;
UPDATE items SET timestamp = datetime(timestamp);
