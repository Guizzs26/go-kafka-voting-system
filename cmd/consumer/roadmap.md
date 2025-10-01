real_time_voting_analysis/
├── cmd/
│   ├── producer/
│   │   └── main.go
│   └── consumer/                <-- NOVO
│       └── main.go              <-- NOVO (Ponto de entrada do Consumer)
├── internal/
│   ├── event/
│   │   ├── publisher.go
│   │   ├── kafka_publisher.go
│   │   ├── consumer.go         <-- NOVO (A interface do nosso Consumer)
│   │   └── kafka_consumer.go   <-- NOVO (A implementação para o Kafka)
│   ├── model/                  # (Você chamou de model, o que está ótimo)
│   │   └── vote.go             # (Reutilizado por ambos os serviços)
│   ├── processing/
│   │   └── vote_processor.go   <-- NOVO (O "cérebro" com a lógica de negócio)
│   └── simulation/
│       └── simulator.go
├── docker-compose.yml
└── go.mod

A Abstração (internal/event/consumer.go): Primeiro, definiremos a interface VoteConsumer. Ela ditará como nosso processador irá "pedir" por novas mensagens, sem se importar de onde elas vêm.

O Cérebro (internal/processing/vote_processor.go): Em seguida, construiremos o VoteProcessor. Ele conterá a lógica de negócio principal (verificar duplicatas, contar votos) e usará a interface VoteConsumer para obter os dados.

A Implementação (internal/event/kafka_consumer.go): Depois, implementaremos a interface VoteConsumer com uma struct KafkaConsumer que saberá se conectar ao Kafka, se juntar a um Consumer Group e ler as mensagens de forma eficiente.

O Maestro (cmd/consumer/main.go): Por último, escreveremos a função main que inicializa tudo, injeta as dependências e gerencia o ciclo de vida do nosso serviço consumidor com um desligamento gracioso.

Essa abordagem garante que nossa lógica de negócio (VoteProcessor) permaneça completamente independente do Kafka.

Pronto para começar a codificar a primeira peça? Vamos definir a interface VoteConsumer no arquivo internal/event/consumer.go.
