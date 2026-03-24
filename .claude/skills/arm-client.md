---
name: arm-client
description: This skill focuses on implementing the Azure Resource Manager (ARM) client using the Azure SDK for Go. It includes creating a client that can authenticate with Azure and interact with various Azure services, particularly for managing Azure Policy resources such as policy definitions and assignments.
--- 


## Purpose
Used when creating the ARM client that will be used to interact with Azure services. 


## instructions
- Implement a client struct that encapsulates the Azure SDK clients and authentication logic.
- Use the Azure SDK for Go to authenticate with Azure and create clients for interacting with Azure Policy resources.
- Ensure that the client can be easily used in the service layer for managing policy definitions and assignments.
- use https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity