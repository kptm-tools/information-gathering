# Information Gathering Service

This microservice is responsible for managing tools that gather information within the Kali Linux ecosystem.

## Information to Gather


### Domain Information Collection

- **Tool:** [ WhoIs ](https://who.is/)
- **Data Scanned:** Domain registration details, 
- **Description:** Information about who owns a domain name, including the owner's contact information and registration dates. Knowing this information can be useful for verifying the legitimacy of a business or understanding its online presence, which can help identify potential risks or vulnerabilities.

### DNS Lookup

- **Tool:** [ DNSLookup ](https://www.nslookup.io/)
- **Data Scanned:** Domain Name System (DNS) records
- **Description:** These are DNS records associated with a website, which help identify how the website is structured online. Understanding these records reveals information about how the site operates, such as where it is hosted and what services are linked to it.

### Subdomain information

- **Tool:** [ theHarvester ](https://github.com/laramies/theHarvester)
- **Data Scanned:** Subdomains and email addresses associated with a domain
- **Description:** This tool collects publicly available information about subdomains and email addresses related to a company. Discovering subdomains can help uncover hidden services that may not be as secure, while email addresses can be targets for phishing attacks. Knowing these can enhance security by identifying potential weak points.


## Testing

To run unit tests, run:

```bash
go test ./...
```
