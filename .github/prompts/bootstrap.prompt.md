The goal is to create a CLI application which can read IP addresses as input, and then determine if that IP address belongs to GitHub or not.

The source of truth for GitHub's IP addresses is the following URL: https://api.github.com/meta
When the program starts, it should fetch this URL and parse the JSON response to get the list of IP addresses.
It should then parse the list of addresses into memory so that it can be easily queried.
This might involve expanding each CIDR range into individual IP addresses, or storing the ranges in a data structure that allows for efficient lookups.
If a match is found for the input IP Address, then the application should also report which sub-section the IP matches (i.e. "hooks", "web", "api", etc).

Ensure that the project's README.md file includes clear instructions on how to install and use the CLI application.
The solution should be implemented in GoLang.