# Regras de Negócio — AskWise

## 1. Regras de Upload de Documentos

### RN-01: Validação de Formato
- **Regra**: Apenas arquivos com extensões `.pdf`, `.csv`, `.txt`, `.yaml`, `.yml`, `.json` e `.md` são aceitos
- **Ação**: Rejeitar upload com HTTP 400 e mensagem indicando formatos permitidos
- **Motivo**: Garantir que o sistema só processe formatos que consegue extrair texto

### RN-02: Limite de Tamanho
- **Regra**: Arquivos devem ter no máximo 10MB
- **Ação**: Rejeitar upload com HTTP 413 e mensagem indicando o limite
- **Motivo**: Evitar processamento excessivo e custos elevados com embeddings na PoC

### RN-03: Arquivo Não Vazio
- **Regra**: Arquivos devem ter conteúdo textual extraível (tamanho > 0 bytes após extração)
- **Ação**: Rejeitar com HTTP 422 e mensagem indicando que o arquivo está vazio ou ilegível
- **Motivo**: Não faz sentido indexar documentos sem conteúdo

### RN-04: Duplicatas
- **Regra**: Se um arquivo com mesmo nome for enviado novamente, o anterior é substituído
- **Ação**: Remover chunks antigos do vector store e reprocessar o novo arquivo
- **Motivo**: Manter a base sempre atualizada com a versão mais recente do documento

### RN-05: Nome Sanitizado
- **Regra**: O nome do arquivo deve ser sanitizado (remover caracteres especiais, limitar tamanho)
- **Ação**: Normalizar o nome antes de armazenar
- **Motivo**: Evitar problemas de segurança e compatibilidade entre sistemas operacionais

## 2. Regras de Processamento (Chunking)

### RN-06: Tamanho de Chunk
- **Regra**: Cada chunk deve ter entre 100 e 500 tokens
- **Ação**: Dividir textos grandes, agrupar textos pequenos
- **Motivo**: Chunks muito grandes diluem a relevância; muito pequenos perdem contexto

### RN-07: Overlap entre Chunks
- **Regra**: Chunks consecutivos devem ter overlap de ~50 tokens
- **Ação**: Incluir final do chunk anterior no início do próximo
- **Motivo**: Evitar perda de informação em quebras de parágrafo

### RN-08: Metadados Obrigatórios
- **Regra**: Todo chunk deve conter: document_id, chunk_index, file_name, file_type
- **Ação**: Incluir metadados ao criar o chunk no vector store
- **Motivo**: Permitir rastreabilidade e citação de fontes

### RN-09: Chunking por Formato
- **Regra**: Cada formato tem estratégia de chunking específica:
  - **PDF/TXT/MD**: Dividir por parágrafos, depois por tamanho se necessário
  - **CSV**: Cada grupo de N linhas vira um chunk (com cabeçalho repetido)
  - **YAML/JSON**: Dividir por chaves de primeiro nível
- **Motivo**: Preservar a estrutura semântica do conteúdo original

## 3. Regras de Busca (Retrieval)

### RN-10: Número de Resultados
- **Regra**: A busca vetorial deve retornar os top 5 chunks mais relevantes
- **Ação**: Buscar top-K=5 por similaridade de cosseno no Qdrant
- **Motivo**: 5 chunks fornecem contexto suficiente sem sobrecarregar o prompt

### RN-11: Score Mínimo de Relevância
- **Regra**: Apenas chunks com score de similaridade >= 0.5 devem ser usados
- **Ação**: Filtrar resultados abaixo do threshold
- **Motivo**: Evitar incluir contexto irrelevante que pode confundir a LLM

### RN-12: Diversidade de Fontes
- **Regra**: Preferir chunks de documentos diferentes quando possível
- **Ação**: Se os top 5 vierem do mesmo documento, expandir a busca
- **Motivo**: Respostas mais completas com múltiplas perspectivas

## 4. Regras de Geração de Resposta (Generation)

### RN-13: Resposta Baseada em Contexto
- **Regra**: A LLM deve responder APENAS com base no contexto fornecido
- **Ação**: Incluir instrução no system prompt: "Responda apenas com base no contexto fornecido.
  Se a informação não estiver no contexto, diga que não encontrou a informação na base."
- **Motivo**: Evitar alucinações (respostas inventadas pela LLM)

### RN-14: Citação de Fontes
- **Regra**: Toda resposta deve incluir referência aos documentos utilizados
- **Ação**: Listar nome do arquivo e chunk ao final da resposta
- **Motivo**: Permitir verificação e confiança na resposta

### RN-15: Idioma da Resposta
- **Regra**: A resposta deve ser no mesmo idioma da pergunta do usuário
- **Ação**: Incluir instrução no system prompt
- **Motivo**: Experiência natural para o usuário

### RN-16: Tom de Resposta
- **Regra**: As respostas devem ser profissionais, diretas e acionáveis
- **Ação**: Incluir instrução no system prompt para responder como um atendente sênior
  experiente que dá respostas claras e passo-a-passo
- **Motivo**: Simular a experiência de consultar um colega sênior

### RN-17: Sem Informação Disponível
- **Regra**: Se nenhum chunk relevante for encontrado, informar ao usuário
- **Ação**: Responder: "Não encontrei informações sobre isso na base de conhecimento.
  Tente reformular a pergunta ou verifique se o documento relevante já foi enviado."
- **Motivo**: Transparência, evitar respostas inventadas

## 5. Regras do Chat (CLI)

### RN-18: Histórico de Conversa
- **Regra**: Manter as últimas 10 mensagens como contexto da conversa
- **Ação**: Incluir histórico no prompt enviado à LLM
- **Motivo**: Permitir perguntas de follow-up ("E sobre o passo 3?" refere-se à resposta anterior)

### RN-19: Comandos Especiais
- **Regra**: O CLI deve suportar comandos iniciados com `/`:
  - `/help` — Mostrar comandos disponíveis
  - `/clear` — Limpar histórico da conversa
  - `/quit` ou `/exit` — Sair do chat
  - `/sources` — Listar documentos indexados
  - `/stats` — Mostrar estatísticas (total docs, chunks, etc.)
- **Motivo**: Controle e navegação para o usuário

### RN-20: Feedback Visual
- **Regra**: O CLI deve mostrar indicadores de progresso durante processamento
- **Ação**: Mostrar "Buscando informações relevantes..." e "Gerando resposta..." durante as etapas
- **Motivo**: O usuário precisa saber que o sistema está trabalhando (LLM pode levar segundos)

## 6. Regras de Dados

### RN-21: Integridade Referencial
- **Regra**: Ao deletar um documento, todos os chunks e embeddings associados devem ser removidos
- **Ação**: Deletar do SQLite e do Qdrant em transação
- **Motivo**: Evitar dados órfãos que poluem a busca

### RN-22: Idempotência de Upload
- **Regra**: Re-upload do mesmo arquivo produz o mesmo resultado (substituição completa)
- **Ação**: Detectar por nome de arquivo, deletar antigo, processar novo
- **Motivo**: Comportamento previsível para o usuário
