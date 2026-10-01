## Why

The received-postcard page is a dead end. A recipient sees the work but cannot open its record or start a postcard of their own. The card also lacks the work's dimensions and holding location, which the approved design places on it (design audit PC-1, PC-2). The composer's character-count copy also differs from the design (PC-4).

## What Changes

- Link the received card's artwork title and a `VIEW IN GALLERY →` action to the work's canonical artwork record.
- Add a `SEND YOUR OWN →` action that opens the composer for the same work, beside `BROWSE THE ARCHIVE →`.
- Show the work's dimensions and holding location on the received card when the record provides them, and omit each line when it does not.
- Make the card's links and the links below it ordinary navigations, so that they work without JavaScript and move the address bar off the bearer URL. The shared site navigation in the page layout is unchanged.
- Align the composer's character-count copy with the design: `MESSAGE — N CHARACTERS LEFT`.

## Capabilities

### Modified Capabilities

- `postcard-sharing`: adds a requirement that the received postcard leads back into the collection and shows the work's record details.

## Coordination

`repair-postcard-experience` remains open. It owns composer rejection handling (2.3), email content (3.2) and its integrated verification (4.x). None of its unfinished tasks covers the received page, so this is a separate change. It adds a new requirement rather than modifying `Postcard delivery and recipient reading are recoverable`, which that change already modifies. The PC-4 copy change touches only the counter wording in the composer and leaves the rejection behaviour of task 2.3 to that change.

## Non-goals

- Changing "Postcard queued" to "Postcard sent" (PC-3). Delivery is asynchronous, so the repository's wording is the truthful one. **Design-side follow-up:** the prototype's confirmation heading (P:2643) and its "Delivered to" copy should adopt the queued wording.
- Changing email content, delivery state, recipient-token handling, or composer validation.

## Impact

- The recipient page handler, the postcard workflow, the artwork record helpers, the postcard Templ page, and the postcard rich-text counter script.
- Adds unit, template and Playwright coverage for the received page.
