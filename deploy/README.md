# Deploy helpers

## Database backups

`backup.sh` dumps the Birdseye MySQL and Dynapay Postgres databases from their containers every
night, keeps 14 days locally and can encrypt (age) and copy them off the server (rclone).

On the VPS, with the repo at `/opt/birdseye`:

```bash
sudo cp deploy/birdseye-backup.service deploy/birdseye-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now birdseye-backup.timer
sudo systemctl start birdseye-backup.service     # run one now
journalctl -u birdseye-backup.service            # check it
```

Optional settings go in `/etc/birdseye-backup.env` (`chmod 600`), see the top of `backup.sh`.
Recommended: an off-site copy (`BACKUP_RCLONE_REMOTE`) encrypted with `BACKUP_AGE_RECIPIENT`.

**Keep `DYNAPAY_ENCRYPTION_KEY` in a password manager.** It decrypts the tenants' M-Pesa
credentials stored in Postgres and is not in the dumps; without it a restored Dynapay database
can't make payments.

Test a restore before launch (commands at the top of `backup.sh`).
