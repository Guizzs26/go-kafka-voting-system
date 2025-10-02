Implementando RegisterVote(ctx, vote)
1. O Objetivo
Este método precisa fazer duas coisas de forma eficiente:

Verificar se um UserID já votou em uma PollID.

Se não votou, registrar que ele votou E incrementar o placar para a OptionID escolhida.

Ele deve retornar true se o voto for novo (válido) e false se for duplicado.

2. As Estruturas de Dados do Redis (O "Porquê")
Para resolver isso, usaremos duas estruturas de dados diferentes do Redis para cada enquete (PollID):

Para saber quem já votou, usaremos um SET.

O que é? Um Set é como uma lista que não permite itens repetidos. É extremamente otimizado para adicionar um item e, principalmente, para verificar se um item já existe. É a ferramenta perfeita para nossa "lista de presença" de votantes.

Nome da Chave: poll:{pollID}:voters (Ex: poll:poll-A-123:voters)

Para guardar o placar, usaremos um HASH.

O que é? Um Hash é como um map dentro de uma única chave do Redis. Ele armazena pares de "campo" -> "valor". É perfeito para guardar o placar, onde o campo é a OptionID e o valor é a contagem de votos.

Nome da Chave: poll:{pollID}:results (Ex: poll:poll-A-123:results)

3. A Estratégia e os Comandos do Redis (O "O Quê")
A forma mais inteligente de verificar e registrar um voto é usando a propriedade do comando SADD.

Comando SADD (Add to Set)

O que faz? Tenta adicionar um membro a um Set.

O Pulo do Gato: Ele retorna o número de membros que foram realmente adicionados. Se você tentar adicionar um UserID que já está no Set, SADD não faz nada e retorna 0. Se o UserID for novo, ele o adiciona e retorna 1.

Função na Biblioteca: rs.client.SAdd(ctx, key, userID)

Isso nos dá uma operação atômica de "verificar e adicionar" em um só passo!

Comando HINCRBY (Increment Hash field by Integer)

O que faz? Incrementa o valor numérico de um campo dentro de um Hash. Se o campo não existir, ele o cria com o valor do incremento. Perfeito para o nosso placar.

Função na Biblioteca: rs.client.HIncrBy(ctx, key, field, increment)

Redis Pipeline (Para Otimização)

Como precisamos executar dois comandos (SADD e HINCRBY) em sequência para um voto válido, a melhor prática é enviá-los juntos para o Redis em um "pacote". Isso se chama Pipeline. Reduz a latência de rede e garante que os comandos sejam executados em ordem.

Funções na Biblioteca: rs.client.Pipeline() para criar um pipeline, e depois pipe.Exec(ctx) para enviar tudo.

4. Sua Lógica Passo a Passo (O "Como")
Dentro do método RegisterVote:

Crie as strings das chaves do Redis com base no vote.PollID (ex: votersKey := fmt.Sprintf("poll:%s:voters", vote.PollID)).

Use o método SAdd para tentar adicionar o vote.UserID ao Set de votantes (votersKey).

Verifique o resultado do SAdd. O método retorna um IntCmd, você pode pegar o valor com .Val().

Se o resultado for 0: Significa que o usuário já estava no Set. É um voto duplicado! Retorne false, nil.

Se o resultado for 1: É um voto novo e válido! Agora, use o método HIncrBy para incrementar o campo vote.OptionID no Hash de resultados (resultsKey) em 1.

Verifique se o HIncrBy retornou algum erro. Se sim, propague o erro.

Se tudo correu bem, retorne true, nil.

Implementando GetResults(ctx, pollID)
1. O Objetivo
Este método é mais simples. Ele precisa buscar o placar completo de uma enquete no Redis e retorná-lo como um map[string]int.

2. A Estrutura de Dados do Redis
Já sabemos: estamos usando um Hash.

3. A Estratégia e o Comando do Redis
Precisamos de um comando que pegue todos os campos e valores de um Hash.

Comando HGETALL (Get all fields and values from Hash)

O que faz? Retorna o mapa completo de um Hash.

Função na Biblioteca: rs.client.HGetAll(ctx, key)

4. Sua Lógica Passo a Passo
Dentro do método GetResults:

Crie a string da chave do Hash de resultados (ex: resultsKey := fmt.Sprintf("poll:%s:results", pollID)).

Use o método HGetAll para buscar os dados.

Verifique o erro. Se houver, retorne o erro.

O resultado de HGetAll será um map[string]string (um mapa de strings para strings, pois o Redis só armazena strings).

Você precisará criar um novo mapa, map[string]int, para ser o retorno da sua função.

Faça um loop (for range) no mapa retornado pelo Redis. Para cada par optionID, countStr:

Converta a string da contagem (countStr) para um inteiro. O pacote strconv tem a função Atoi (strconv.Atoi(countStr)) para isso. Lembre-se de verificar o erro da conversão!

Adicione o par optionID, countInt ao seu novo mapa.

Retorne o mapa de inteiros preenchido.

