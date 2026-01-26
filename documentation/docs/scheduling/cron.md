# Cron Expressions

Detailed guide to cron expression syntax.

## Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

## Field Values

| Field | Values | Special Characters |
|-------|--------|-------------------|
| Minute | 0-59 | `, - * /` |
| Hour | 0-23 | `, - * /` |
| Day of Month | 1-31 | `, - * / L W` |
| Month | 1-12 or JAN-DEC | `, - * /` |
| Day of Week | 0-6 or SUN-SAT | `, - * / L #` |

## Special Characters

### Asterisk (*)

Match any value:

```
* * * * *     # Every minute
0 * * * *     # Every hour at minute 0
```

### Comma (,)

List multiple values:

```
0 9,17 * * *  # 9 AM and 5 PM
0 0 1,15 * *  # 1st and 15th of month
```

### Hyphen (-)

Range of values:

```
0 9-17 * * *  # Every hour 9 AM to 5 PM
0 0 * * 1-5   # Mon through Fri at midnight
```

### Slash (/)

Step values:

```
*/15 * * * *  # Every 15 minutes
0 */2 * * *   # Every 2 hours
```

### L (Last)

Last day:

```
0 0 L * *     # Last day of month
0 0 * * 5L    # Last Friday of month
```

### W (Weekday)

Nearest weekday:

```
0 0 15W * *   # Nearest weekday to 15th
0 0 1W * *    # First weekday of month
```

### Hash (#)

Nth occurrence:

```
0 0 * * 5#3   # Third Friday of month
0 0 * * 1#1   # First Monday of month
```

## Examples

### Time-Based

| Cron | Description |
|------|-------------|
| `* * * * *` | Every minute |
| `*/5 * * * *` | Every 5 minutes |
| `0 * * * *` | Every hour |
| `0 */2 * * *` | Every 2 hours |
| `0 0 * * *` | Daily at midnight |
| `0 9 * * *` | Daily at 9 AM |
| `0 9,18 * * *` | 9 AM and 6 PM |
| `0 9-17 * * *` | Every hour 9 AM-5 PM |
| `30 9 * * *` | Daily at 9:30 AM |

### Day-Based

| Cron | Description |
|------|-------------|
| `0 0 * * 0` | Every Sunday midnight |
| `0 9 * * 1` | Every Monday 9 AM |
| `0 9 * * 1-5` | Weekdays 9 AM |
| `0 9 * * 0,6` | Weekends 9 AM |
| `0 0 1 * *` | 1st of month midnight |
| `0 0 1,15 * *` | 1st and 15th |
| `0 0 L * *` | Last day of month |

### Month-Based

| Cron | Description |
|------|-------------|
| `0 0 1 1 *` | January 1st |
| `0 0 1 */3 *` | Every 3 months |
| `0 0 1 1,4,7,10 *` | Quarterly |
| `0 0 15 * *` | 15th of each month |

### Complex

| Cron | Description |
|------|-------------|
| `0 9 * * 1#1` | First Monday 9 AM |
| `0 9 * * 5L` | Last Friday 9 AM |
| `0 9 1W * *` | First weekday 9 AM |
| `0 0 1-7 * 1` | First Monday of month |
| `0 */4 * * 1-5` | Every 4 hours weekdays |

## Business Scenarios

### Daily Reports

```
# Daily at 9 AM
0 9 * * *

# Weekdays only at 9 AM
0 9 * * 1-5
```

### Weekly Tasks

```
# Weekly Sunday maintenance
0 2 * * 0

# Weekly Monday standup reminder
0 8:55 * * 1
```

### Monthly Tasks

```
# Monthly invoice on 1st
0 9 1 * *

# End of month report
0 17 L * *
```

### Periodic Checks

```
# Health check every 5 minutes
*/5 * * * *

# Data sync every hour
0 * * * *
```

## Validation

### Check Syntax

```bash
m9m schedule validate "0 9 * * 1-5"
```

### Preview Runs

```bash
m9m schedule preview "0 9 * * 1-5" --count 10
```

Output:

```
Next 10 runs:
1. Mon, 15 Jan 2024 09:00:00 UTC
2. Tue, 16 Jan 2024 09:00:00 UTC
3. Wed, 17 Jan 2024 09:00:00 UTC
...
```

## Common Mistakes

### Incorrect: Every Second

```
# WRONG - cron doesn't support seconds
* * * * * *

# RIGHT - minimum is every minute
* * * * *
```

### Incorrect: Both Day Fields

Setting both day-of-month and day-of-week can be confusing:

```
# Runs on 15th AND every Monday (OR logic)
0 9 15 * 1

# For 15th only if it's Monday, use code logic
```

### Incorrect: Step with Range

```
# WRONG
0 9-17/2 * * *

# RIGHT - step on full range
0 */2 * * *
# Or list specific hours
0 9,11,13,15,17 * * *
```

## Online Tools

Test expressions before deploying:

- crontab.guru - Visual cron editor
- cronitor.io - Cron expression generator

## Extended Syntax

Some systems support 6-field cron (with seconds):

```
┌───────────── second (0-59)
│ ┌───────────── minute (0-59)
│ │ ┌───────────── hour (0-23)
│ │ │ ┌───────────── day of month (1-31)
│ │ │ │ ┌───────────── month (1-12)
│ │ │ │ │ ┌───────────── day of week (0-6)
│ │ │ │ │ │
* * * * * *
```

m9m uses standard 5-field cron by default. Configure 6-field:

```yaml
scheduling:
  cronFormat: "6-field"
```

## Named Schedules

Use predefined names instead of expressions:

| Name | Equivalent |
|------|------------|
| `@yearly` | `0 0 1 1 *` |
| `@monthly` | `0 0 1 * *` |
| `@weekly` | `0 0 * * 0` |
| `@daily` | `0 0 * * *` |
| `@hourly` | `0 * * * *` |

Example:

```json
{
  "cronExpression": "@daily"
}
```
