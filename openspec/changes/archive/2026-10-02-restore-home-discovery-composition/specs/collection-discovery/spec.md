## MODIFIED Requirements

### Requirement: Home gives an unfamiliar visitor clear routes into the collection

The system SHALL render the home page with the collection argument, work of the day, collection counts, recent additions, and distinct discovery routes. The hero SHALL offer exactly two actions: BROWSE ARTISTS → to the artist index as the primary action, and COMPARE TWO WORKS → to Dual Mode. The recent-additions header SHALL link to the artwork index as ALL WORKS →. The page SHALL present three feature cards in this order: TWO WINDOWS, linking to Dual Mode; TIMELINE, linking to the Timeline; and POSTCARD SERVICE, linking to the postcard entry point and to Contributors. The count strip SHALL label its live counts REPRODUCTIONS, ARTISTS and SCHOOLS, and SHALL state the collection's period, from the 3rd century to the early 20th, labelled CENTURY. Postcard copy SHALL NOT describe period music as a sender option.

#### Scenario: Visitor opens the home page

- **WHEN** a regular visitor opens the home route
- **THEN** they can identify the collection and navigate to artist browsing, artwork browsing, Dual Mode, the Timeline, postcards, and Contributors without entering a search term, and reach Inspiration and Guided Tours through the primary navigation.

#### Scenario: Visitor follows a feature card

- **WHEN** a visitor activates OPEN DUAL MODE →, OPEN THE TIMELINE →, SEND A POSTCARD →, or MEET THE CONTRIBUTORS →
- **THEN** the link leads to the real server-rendered Dual Mode, Timeline, postcard, or Contributors route respectively.
