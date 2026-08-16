FROM node:22-alpine

RUN npm install --global firebase-tools

WORKDIR /app

COPY firebase.json ./

ENV CI=true

EXPOSE 9099 4000

CMD ["firebase", "emulators:start", "--only", "auth", "--project", "demo-crown-and-borough"]
