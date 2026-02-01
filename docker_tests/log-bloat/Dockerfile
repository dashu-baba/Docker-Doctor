FROM alpine:latest

# Install bash for scripting
RUN apk add --no-cache bash

# Copy the log generator script
COPY log-generator.sh /log-generator.sh
RUN chmod +x /log-generator.sh

# Run the script that generates logs
CMD ["/log-generator.sh"]