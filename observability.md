A minha contraproposta é dividir a sua sugestão em duas fases:

Fase de Métricas 1: A Instrumentação (O "Relógio de Pulso")
Nesta fase, nós focamos apenas no código Go. Nós adicionamos a biblioteca de cliente do Prometheus (prometheus/client_golang) à nossa aplicação consumer e começamos a registrar as métricas que nos interessam:

votos_processados_total: Um contador de votos válidos.
votos_duplicados_total: Um contador de fraudes detectadas.
tempo_processamento_voto_seconds: Um histograma para medir a latência.
kafka_consumer_group_lag: A métrica mais importante do Kafka, que mede o "atraso" do consumidor em relação ao produtor.

O resultado desta fase é que nossa aplicação consumer passará a expor um endpoint http://localhost:PORTA/metrics com todos esses dados em tempo real. Nós ainda não teremos um dashboard, mas já teremos os dados brutos. É como ter um relógio de pulso preciso antes de construir uma central de controle completa.


---------- // ----------

Fase de Métricas 2: A Visualização (A "Central de Controle")
Esta é a implementação completa da sua ideia. Com a aplicação já "instrumentada", nós adicionamos os contêineres do Prometheus e do Grafana ao nosso docker-compose.yml. O Prometheus vai "raspar" (scrape) os dados do endpoint /metrics e o Grafana vai nos permitir criar os dashboards para visualizar tudo de forma gráfica.